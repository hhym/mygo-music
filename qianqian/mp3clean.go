package qianqian

import (
	"io"
)

/* 网络电台的直播流不干净：
   - 蜻蜓FM 这类会在开头塞一段随机 junk（实测前 140 字节完全不是音频）
   - icy-metadata: 1 的流会在音频中间插入元数据块
   直接透传给 <audio> 会「这首歌无法播放」。
   这里做一个帧级净化器：扫描 MPEG 帧头 → 校验 → 按帧长整帧输出，
   非音频字节一律丢弃。这样输出就是一条干净的 MP3 流。 */

var brTableV1L3 = [16]int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0}
var brTableV2L3 = [16]int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0}
var srTableV1 = [4]int{44100, 48000, 32000, 0}
var srTableV2 = [3]int{22050, 24000, 16000}
var srTableV25 = [3]int{11025, 12000, 8000}

// mp3FrameLen 返回这一帧的总字节数；0 表示这里不是合法帧头
func mp3FrameLen(b []byte) int {
	if len(b) < 4 {
		return 0
	}
	if b[0] != 0xFF || b[1]&0xE0 != 0xE0 {
		return 0
	}
	ver := (b[1] >> 3) & 3   // 0=MPEG2.5 1=保留 2=MPEG2 3=MPEG1
	layer := (b[1] >> 1) & 3 // 1=Layer III
	if ver == 1 || layer != 1 {
		return 0 // 只处理 Layer III
	}
	brIdx := int(b[2] >> 4)
	srIdx := int((b[2] >> 2) & 3)
	pad := int((b[2] >> 1) & 1)
	if brIdx == 0 || brIdx == 15 || srIdx == 3 {
		return 0
	}

	var br int
	if ver == 3 {
		br = brTableV1L3[brIdx]
	} else {
		br = brTableV2L3[brIdx]
	}
	if br == 0 {
		return 0
	}

	var sr int
	switch ver {
	case 3:
		sr = srTableV1[srIdx]
	case 2:
		sr = srTableV2[srIdx]
	default:
		sr = srTableV25[srIdx]
	}
	if sr == 0 {
		return 0
	}

	samples := 1152
	if ver != 3 {
		samples = 576
	}
	// 帧长 = 每帧采样数 / 8 * 码率(kbps) * 1000 / 采样率 + padding
	n := (samples/8)*br*1000/sr + pad
	if n < 24 || n > 2200 {
		return 0
	}
	return n
}

type flusher interface{ Flush() }

// copyMP3Clean 把 src 净化后写入 dst。遇到非音频字节就地跳过，遇到坏帧丢弃。
func copyMP3Clean(dst io.Writer, src io.Reader, f flusher) error {
	const maxHold = 48 * 1024 // 缓冲区上限，防止找不到帧时无限增长
	buf := make([]byte, 0, 8192)
	tmp := make([]byte, 4096)

	write := func(p []byte) error {
		_, err := dst.Write(p)
		if err == nil && f != nil {
			f.Flush()
		}
		return err
	}

	for {
		n, rerr := src.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}

		for {
			if len(buf) < 4 {
				break
			}
			// 向前扫到第一个合法帧头
			i := 0
			for i <= len(buf)-4 && mp3FrameLen(buf[i:]) == 0 {
				i++
			}
			if i > 0 {
				buf = buf[i:]
				if len(buf) < 4 {
					break
				}
			}
			fl := mp3FrameLen(buf)
			if fl == 0 {
				break
			}
			if len(buf) < fl {
				break // 等更多数据
			}
			if err := write(buf[:fl]); err != nil {
				return err
			}
			buf = buf[fl:]
		}

		// 长时间找不到帧说明这段流可能不是 Layer III，限制住内存
		if len(buf) > maxHold {
			buf = buf[len(buf)-16*1024:]
		}
		if rerr != nil {
			return rerr
		}
	}
}
