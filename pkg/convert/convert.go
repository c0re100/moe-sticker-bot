package convert

import (
	"bufio"
	"errors"
	"image"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var FFMPEG_BIN = "ffmpeg"
var BSDTAR_BIN = "bsdtar"
var CONVERT_BIN = "convert"
var CONVERT_ARGS []string

// See: http://en.wikipedia.org/wiki/Binary_prefix
const (
	// Decimal
	KB = 1000
	MB = 1000 * KB
	GB = 1000 * MB
	TB = 1000 * GB
	PB = 1000 * TB

	// Binary
	KiB = 1024
	MiB = 1024 * KiB
	GiB = 1024 * MiB
	TiB = 1024 * GiB
	PiB = 1024 * TiB
)

// Should call before using functions in this package.
// Otherwise, defaults to Linux environment.
// This function also call CheckDeps to check if executables exist and return a string slice
// containing binaries that are not found in PATH.
func InitConvert() {
	switch runtime.GOOS {
	case "linux":
		CONVERT_BIN = "convert"
	default:
		CONVERT_BIN = "magick"
		CONVERT_ARGS = []string{"convert"}
	}
	unfoundBins := CheckDeps()
	if len(unfoundBins) != 0 {
		log.Warning("Following required executables not found!:")
		log.Warnln(strings.Join(unfoundBins, "  "))
		log.Warning("Please install missing executables to your PATH, or some features will not work!")
	}
}

func CheckDeps() []string {
	unfoundBins := []string{}

	if _, err := exec.LookPath(FFMPEG_BIN); err != nil {
		unfoundBins = append(unfoundBins, FFMPEG_BIN)
	}
	if _, err := exec.LookPath(BSDTAR_BIN); err != nil {
		unfoundBins = append(unfoundBins, BSDTAR_BIN)
	}
	if _, err := exec.LookPath(CONVERT_BIN); err != nil {
		unfoundBins = append(unfoundBins, CONVERT_BIN)
	}
	if _, err := exec.LookPath("exiv2"); err != nil {
		unfoundBins = append(unfoundBins, "exiv2")
	}
	if _, err := exec.LookPath("gifsicle"); err != nil {
		unfoundBins = append(unfoundBins, "gifsicle")
	}
	return unfoundBins
}

func getImageWH(f string) (int, int) {
	input, _ := os.Open(f)
	reader := bufio.NewReader(input)
	config, _, _ := image.DecodeConfig(reader)
	return config.Width, config.Height
}

func IMToWebp(f string) (string, error) {
	w, h := getImageWH(f)
	pathOut := f + ".webp"
	bin := "waifu2x-caffe-cui"
	var args []string
	if w >= h {
		args = append(args, "-i", f, "-m", "noise_scale", "-w", "512", "--noise_level", "3", "-o", pathOut)
	} else {
		args = append(args, "-i", f, "-m", "noise_scale", "-h", "512", "--noise_level", "3", "-o", pathOut)
	}

	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		log.Warnln("imToWebp ERROR:", string(out))
		return "", err
	}

	if st, _ := os.Stat(pathOut); st.Size() > 255*KiB {
		args := CONVERT_ARGS
		args = append(args, "-resize", "512x512", "-filter", "Lanczos", f+"[0]", pathOut)
		exec.Command(bin, args...).CombinedOutput()
	}
	return pathOut, err
}

func IMToWebpWA(f string) error {
	pathOut := f
	bin := CONVERT_BIN
	args := CONVERT_ARGS
	qualities := []string{"75", "50"}
	for _, q := range qualities {
		args = append(args, "-define", "webp:quality="+q,
			"-resize", "512x512", "-gravity", "center", "-extent", "512x512",
			"-background", "none", f+"[0]", pathOut)

		out, err := exec.Command(bin, args...).CombinedOutput()
		if err != nil {
			log.Warnln("imToWebp ERROR:", string(out))
			return err
		}
		if st, _ := os.Stat(pathOut); st.Size() > 300*KiB {
			continue
		} else {
			return nil
		}
	}
	return errors.New("bad webp")
}

func IMToPng(f string) (string, error) {
	pathOut := f + ".png"
	bin := CONVERT_BIN
	args := CONVERT_ARGS
	args = append(args, f, pathOut)

	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		log.Warnln("imToPng ERROR:", string(out))
		return "", err
	}
	return pathOut, err
}

// func IMToGIF(f string) (string, error) {
// 	if strings.HasSuffix(f, ".webm") {
// 		f2, _ := FFToAPNG(f)
// 		f = "apng:" + f2
// 	}

// 	pathOut := f + ".gif"
// 	bin := CONVERT_BIN
// 	args := CONVERT_ARGS
// 	args = append(args, "-layers", "optimize", f, pathOut)

// 	out, err := exec.Command(bin, args...).CombinedOutput()
// 	if err != nil {
// 		log.Warnln("imToGIF ERROR:", string(out))
// 		return "", err
// 	}
// 	return pathOut, err
// }

func IMToApng(f string) (string, error) {
	pathOut := f + ".apng"
	bin := CONVERT_BIN
	args := CONVERT_ARGS
	args = append(args, f, pathOut)

	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		log.Warnln("imToApng ERROR:", string(out))
		return "", err
	}
	return pathOut, err
}

func FFToWebm(f string) (string, error) {
	pathOut := f + ".webm"
	bin := FFMPEG_BIN
	baseargs := []string{"-hide_banner", "-i", f,
		"-vf", "scale=512:512:force_original_aspect_ratio=decrease:flags=lanczos", "-pix_fmt", "yuva420p",
		"-c:v", "libvpx-vp9", "-cpu-used", "5",
	}

	for rc := 0; rc < 4; rc++ {
		rcargs := []string{}
		switch rc {
		case 0:
			rcargs = []string{"-minrate", "50k", "-b:v", "350k", "-maxrate", "450k"}
		case 1:
			rcargs = []string{"-minrate", "50k", "-b:v", "200k", "-maxrate", "300k"}
		case 2:
			rcargs = []string{"-minrate", "20k", "-b:v", "100k", "-maxrate", "200k"}
		case 3:
			rcargs = []string{"-minrate", "10k", "-b:v", "50k", "-maxrate", "100k"}
		}
		args := append(baseargs, rcargs...)
		args = append(args, []string{"-to", "00:00:03", "-an", "-y", pathOut}...)
		out, err := exec.Command(bin, args...).CombinedOutput()
		if err != nil {
			log.Warnln("ffToWebm ERROR:", string(out))
			//FFMPEG does not support animated webp.
			//Convert to APNG first than WEBM.
			if strings.Contains(string(out), "skipping unsupported chunk: ANIM") {
				log.Warnln("Trying to convert to APNG first.")
				f2, _ := IMToApng(f)
				return FFToWebm(f2)
			}
			return pathOut, err
		}
		if stat, _ := os.Stat(pathOut); stat.Size() > 255*KiB {
			continue
		} else {
			return pathOut, err
		}
	}
	log.Errorln("fftowebm unable to compress below 256KiB:", pathOut)
	return pathOut, errors.New("fftowebm unable to compress below 256KiB")
}

// This function will be called if Telegram's API rejected our webm.
func FFToWebmSafe(f string) (string, error) {
	pathOut := f + ".webm"
	bin := FFMPEG_BIN
	args := []string{"-hide_banner", "-i", f,
		"-vf", "scale=512:512:force_original_aspect_ratio=decrease:flags=lanczos", "-pix_fmt", "yuva420p",
		"-c:v", "libvpx-vp9", "-cpu-used", "5", "-minrate", "50k", "-b:v", "200k", "-maxrate", "300k",
		"-to", "00:00:02.800", "-r", "30", "-an", "-y", pathOut}

	cmd := exec.Command(bin, args...)
	err := cmd.Run()
	return pathOut, err
}

func FFToGif(f string) (string, error) {
	var decoder []string
	var args []string
	if strings.HasSuffix(f, ".webm") {
		decoder = []string{"-c:v", "libvpx-vp9"}
	}
	pathOut := f + ".gif"
	bin := FFMPEG_BIN
	args = append(args, decoder...)
	args = append(args, "-i", f, "-hide_banner",
		"-lavfi", "split[a][b];[a]palettegen[p];[b][p]paletteuse=dither=atkinson",
		"-gifflags", "-transdiff", "-gifflags", "-offsetting",
		"-loglevel", "error", "-y", pathOut)

	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		log.Warnf("ffToGif ERROR:\n%s", string(out))
		return "", err
	}
	//Optimize GIF produced by ffmpeg
	exec.Command("gifsicle", "--batch", "-O2", "--lossy=60", pathOut).CombinedOutput()

	return pathOut, err
}

func FFToAPNG(f string) (string, error) {
	var decoder []string
	var args []string
	if strings.HasSuffix(f, ".webm") {
		decoder = []string{"-c:v", "libvpx-vp9"}
	}
	pathOut := f + ".apng"
	bin := FFMPEG_BIN
	args = append(args, decoder...)
	args = append(args, "-i", f, "-hide_banner",
		"-loglevel", "error", "-y", pathOut)

	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		log.Warnf("ffToAPNG ERROR:\n%s", string(out))
		return "", err
	}
	return pathOut, err
}

func IMStackToWebp(base string, overlay string) (string, error) {
	bin := CONVERT_BIN
	args := CONVERT_ARGS
	fOut := base + ".composite.webp"

	args = append(args, base, overlay, "-background", "none", "-filter", "Lanczos", "-resize", "512x512", "-composite",
		"-define", "webp:lossless=true", fOut)
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		log.Errorln("IM stack ERROR!", string(out))
		return "", err
	} else {
		return fOut, nil
	}
}

func RlottieToWebm(f string) (string, error) {
	bin := "msb_rlottie.py"
	fOut := f + ".apng"
	args := []string{f, fOut, "75"}
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		log.Errorln("lottieToGIF ERROR!", string(out))
		return "", err
	}
	return FFToWebm(fOut)
}

// Replaces tgs to gif.
func RlottieToGIF(f string) (string, error) {
	bin := "msb_rlottie.py"
	fOut := strings.ReplaceAll(f, ".tgs", ".gif")
	args := []string{f, fOut, "75"}
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		log.Errorln("lottieToGIF ERROR!", string(out))
		return "", err
	}
	//Optimize GIF
	exec.Command("gifsicle", "--batch", "-O2", "--lossy=60", fOut).CombinedOutput()
	return fOut, nil
}

// Replaces tgs to webp.
// The only purpose for this func is for WhatsApp export.
func RlottieToWebp(f string) (string, error) {
	bin := "msb_rlottie.py"
	pathOut := strings.ReplaceAll(f, ".tgs", ".webp")

	qualities := []string{"50", "20", "0"}
	for _, q := range qualities {
		args := []string{f, pathOut, q}
		out, err := exec.Command(bin, args...).CombinedOutput()
		if err != nil {
			log.Errorln("RlottieToWebp ERROR!", string(out))
			return "", err
		}
		//WhatsApp uses KiB.
		if st, _ := os.Stat(pathOut); st.Size() > 500*KiB {
			log.Warnf("convert: awebp exceeded 500KiB, q:%s z:%d s:%s", q, st.Size(), pathOut)
			continue
		} else {
			return pathOut, nil
		}
	}
	log.Warnln("all quality failed! s:", pathOut)
	return pathOut, errors.New("bad animated webp?")
}

// Replaces .webm ext to .webp
func IMToAnimatedWebpLQ(f string) error {
	pathOut := strings.ReplaceAll(f, ".webm", ".webp")
	bin := CONVERT_BIN
	args := CONVERT_ARGS
	args = append(args, "-resize", "128x128", f, pathOut)

	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		log.Warnln("imToWebp ERROR:", string(out))
		return err
	}
	return err
}

// Replaces .webm ext to .webp
func FFToAnimatedWebpLQ(f string) error {
	pathOut := strings.ReplaceAll(f, ".webm", ".webp")
	bin := FFMPEG_BIN

	args := []string{"-hide_banner", "-c:v", "libvpx-vp9", "-i", f,
		"-vf", "scale=128:128:force_original_aspect_ratio=decrease",
		"-loop", "0", "-pix_fmt", "yuva420p",
		"-an", "-y", pathOut}

	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		log.Warnln("ffToAnimatedWebpWA ERROR:", string(out))
		return err
	}
	return nil
}

// // animated webp has a pretty bad compression ratio comparing to VP9,
// // shrink down quality as much as possible.
func FFToAnimatedWebpWA(f string) error {
	pathOut := strings.ReplaceAll(f, ".webm", ".webp")
	bin := FFMPEG_BIN
	//Try qualities from best to worst.
	qualities := []string{"75", "50", "20", "0", "_DS256", "_DS256Q0"}

	for _, q := range qualities {
		args := []string{"-hide_banner", "-c:v", "libvpx-vp9", "-i", f,
			"-vf", "scale=512:512:force_original_aspect_ratio=decrease,pad=512:512:-1:-1:color=black@0",
			"-quality", q, "-loop", "0", "-pix_fmt", "yuva420p",
			"-an", "-y", pathOut}

		if q == "_DS256" {
			args = []string{"-hide_banner", "-c:v", "libvpx-vp9", "-i", f,
				"-vf", "scale=256:256:force_original_aspect_ratio=decrease,pad=512:512:-1:-1:color=black@0",
				"-quality", "20", "-loop", "0", "-pix_fmt", "yuva420p",
				"-an", "-y", pathOut}
		}

		if q == "_DS256Q0" {
			args = []string{"-hide_banner", "-c:v", "libvpx-vp9", "-i", f,
				"-vf", "scale=256:256:force_original_aspect_ratio=decrease,pad=512:512:-1:-1:color=black@0",
				"-quality", "0", "-loop", "0", "-pix_fmt", "yuva420p",
				"-an", "-y", pathOut}
		}

		out, err := exec.Command(bin, args...).CombinedOutput()
		if err != nil {
			log.Warnln("ffToAnimatedWebpWA ERROR:", string(out))
			return err
		}
		//WhatsApp uses KiB.
		if st, _ := os.Stat(pathOut); st.Size() > 500*KiB {
			log.Warnf("convert: awebp exceeded 500KiB, q:%s z:%d s:%s", q, st.Size(), pathOut)
			continue
		} else {
			return nil
		}
	}
	log.Warnln("all quality failed! s:", pathOut)

	return errors.New("bad animated webp?")
}

// appends png
func FFtoPNG(f string, pathOut string) error {
	var args []string
	bin := FFMPEG_BIN
	args = append(args, "-c:v", "libvpx-vp9", "-i", f, "-hide_banner",
		"-loglevel", "error", "-frames", "1", "-y", pathOut)

	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		log.Warnf("fftoPNG ERROR:\n%s", string(out))
		return err
	}
	return err
}

// Replaces .webm or .webp to .png
func IMToPNGThumb(f string) error {
	pathOut := strings.ReplaceAll(f, ".webm", ".png")
	pathOut = strings.ReplaceAll(pathOut, ".webp", ".png")

	if strings.HasSuffix(f, ".webm") {
		tempThumb := f + ".thumb.png"
		FFtoPNG(f, tempThumb)
		f = tempThumb
	}

	bin := CONVERT_BIN
	args := CONVERT_ARGS
	args = append(args,
		"-resize", "96x96",
		"-gravity", "center", "-extent", "96x96", "-background", "none",
		f+"[0]", pathOut)

	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		log.Warnln("imToPng ERROR:", string(out))
		return err
	}
	return err
}

func SetImageTime(f string, t time.Time) error {
	return os.Chtimes(f, t, t)
	// asciiTime := t.Format("2006:01:02 15:04:05")
	// _, err := exec.Command("exiv2", "-M", "set Exif.Image.DateTime "+asciiTime, f).CombinedOutput()
	// if err != nil {
	// 	return err
	// }
	// return nil
}
