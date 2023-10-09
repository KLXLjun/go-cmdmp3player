package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"strings"
	"time"
	"unicode"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/gdamore/tcell"
)

// 扫描目录
var filepath string = "/home/huyaoi/音乐/Sdcard/Music"

// 绘制行
func drawTextLine(screen tcell.Screen, x, y int, s string, style tcell.Style) {
	for _, r := range s {
		screen.SetContent(x, y, r, nil, style)
		if len(string(r)) == 3 || len(string(r)) == 2 {
			x++
			screen.SetContent(x, y, rune('_'), nil, style)
			x++
		} else {
			x++
		}
	}
}

var nowdispindex int = 0           //当前指向的歌曲
var nowdispmaxindex int = 0        //最大歌曲数量
var listarray = make([]listrow, 0) //歌曲列表
var tip string = ""                //提示

var nowplayindex = -1 //当前播放歌曲位于本页的下标
var nowplaytitle = "" //当前播放歌曲标题
var nowlrc = ""       //当前歌词

var maxpagerow = 10 //每页最大

// 音频引擎
var ap *audioPanel
var fsteam *os.File
var ssc beep.StreamSeekCloser
var bformat beep.Format

var lrcEngine []lrcStruct = nil

type listrow struct {
	name     string
	filename string
}

type audioPanel struct {
	sampleRate beep.SampleRate
	streamer   beep.StreamSeeker
	ctrl       *beep.Ctrl
	resampler  *beep.Resampler
	volume     *effects.Volume
}

// 音频面板
func newAudioPanel(sampleRate beep.SampleRate, streamer beep.StreamSeeker) *audioPanel {
	ctrl := &beep.Ctrl{Streamer: beep.Loop(-1, streamer)}
	resampler := beep.ResampleRatio(4, 1, ctrl)
	volume := &effects.Volume{Streamer: resampler, Base: 2}
	return &audioPanel{sampleRate, streamer, ctrl, resampler, volume}
}

func (ap *audioPanel) play() {
	speaker.Play(ap.volume)
}

// 画面绘制
func TDrawCls(screen tcell.Screen) {
	//主页背景样式
	mainBackgroundStyle := tcell.StyleDefault.
		Foreground(tcell.NewHexColor(0xD7D8A2))

	//主页样式
	mainStyle := tcell.StyleDefault.
		Foreground(tcell.ColorGreen)

	//播放样式
	mainPlayStyle := tcell.StyleDefault.
		Foreground(tcell.ColorRed)

	//选择样式
	selectStyle := tcell.StyleDefault.
		Background(tcell.ColorWhiteSmoke).
		Foreground(tcell.ColorGreen)

	//选择并正在播放样式
	PlaySelectStyle := tcell.StyleDefault.
		Background(tcell.ColorWhiteSmoke).
		Foreground(tcell.ColorRed)

	//提示样式
	tipStyle := tcell.StyleDefault.
		Foreground(tcell.ColorGreen)

	//清空
	screen.Fill(' ', mainBackgroundStyle)

	//获取所有页数
	allPage := int(math.Ceil(float64(nowdispmaxindex) / float64(maxpagerow)))

	//获取选择的项的页数
	selectIndexPage := int(math.Ceil(float64(nowdispindex+1) / float64(maxpagerow)))

	nowPageMaxRow := 0
	if selectIndexPage < allPage {
		nowPageMaxRow = maxpagerow
	} else {
		nowPageMaxRow = nowdispmaxindex - ((selectIndexPage - 1) * maxpagerow)
	}

	//绘制页数和总项目数
	drawTextLine(screen, 5, 1, fmt.Sprintf("%d/%d [%d/%d]", selectIndexPage, allPage, nowdispindex+1, nowdispmaxindex), mainBackgroundStyle) //当前显示页数/最大页数

	//绘制列表
	for i := 0; i < nowPageMaxRow; i++ {
		var nowFirstIndex = (selectIndexPage - 1) * maxpagerow
		if nowFirstIndex+i == nowdispindex {
			//判断是否是正在播放的
			if nowFirstIndex+i == nowplayindex {
				drawTextLine(screen, 0, 2+i, " >>> ", PlaySelectStyle)
				drawTextLine(screen, 5, 2+i, listarray[nowFirstIndex+i].name, PlaySelectStyle)
			} else {
				drawTextLine(screen, 0, 2+i, " --> ", PlaySelectStyle)
				drawTextLine(screen, 5, 2+i, listarray[nowFirstIndex+i].name, selectStyle)
			}
		} else {
			//未选择项目
			if nowFirstIndex+i == nowplayindex {
				drawTextLine(screen, 0, 2+i, " >>> ", mainPlayStyle)
				drawTextLine(screen, 5, 2+i, listarray[nowFirstIndex+i].name, mainPlayStyle)
			} else {
				drawTextLine(screen, 0, 2+i, "     ", mainPlayStyle)
				drawTextLine(screen, 5, 2+i, listarray[nowFirstIndex+i].name, mainStyle)
			}
		}
	}

	drawTextLine(screen, 5, 30, tip, tipStyle)          //绘制提示
	drawTextLine(screen, 5, 16, nowplaytitle, tipStyle) //绘制当前播放歌曲名称
	drawTextLine(screen, 5, 17, nowlrc, tipStyle)       //绘制当前播放歌曲的歌词

	//处理与绘制播放时长
	var positionStatus string = ""
	if ap != nil {
		position := ap.sampleRate.D(ap.streamer.Position())
		length := ap.sampleRate.D(ap.streamer.Len())
		positionStatus = fmt.Sprintf("%v / %v", fmtDuration(position), fmtDuration(length))
	} else {
		positionStatus = "00:00:00 / 00:00:00"
	}
	drawTextLine(screen, 5, 18, positionStatus, mainBackgroundStyle)

	//绘制音量
	var volumeStatus string = ""
	if ap != nil {
		volume := ap.volume.Volume
		volumeStatus = fmt.Sprintf("音量: %.1f", volume)
	} else {
		volumeStatus = "音量: 0.0"
	}
	drawTextLine(screen, 5, 19, volumeStatus, mainBackgroundStyle)

	//绘制速度
	var speedStatus string = ""
	if ap != nil {
		speed := ap.resampler.Ratio()
		speedStatus = fmt.Sprintf("速度: %.3fx", speed)
	} else {
		speedStatus = "速度: 0.000x"
	}
	drawTextLine(screen, 5, 20, speedStatus, mainBackgroundStyle)

	//绘制播放状态
	var state string = "-"
	if ap != nil {
		if ap.ctrl.Paused {
			state = "暂停"
		} else {
			state = "播放"
		}
	} else {
		state = "停止"
	}
	drawTextLine(screen, 5, 21, "播放状态: "+state, mainBackgroundStyle)
}

// 按键事件响应
func THandle(event tcell.Event) (change, exit bool) {
	switch event := event.(type) {
	case *tcell.EventKey:
		//处理退出
		if event.Key() == tcell.KeyESC {
			return false, true
		}

		//播放选中
		if event.Key() == tcell.KeyEnter {
			tip = string("播放")
			if ssc != nil {
				ssc.Close()
				ssc = nil
			}
			if fsteam != nil {
				fsteam.Close()
				fsteam = nil
			}

			lrcPath := path.Join(filepath, GetFileName(listarray[nowdispindex].filename)+".lrc")
			nowlrc = lrcPath
			if ok, _ := PathExists(lrcPath); ok {
				if ok, bk := loadLyric(ReadFile(lrcPath)); ok {
					lrcEngine = bk
				} else {
					lrcEngine = nil
				}
			} else {
				lrcEngine = nil
			}

			var errs error
			fsteam, errs = os.Open(filepath + "/" + listarray[nowdispindex].filename)
			if errs != nil {
				report(errs)
			}

			nowplayindex = nowdispindex

			var errs2 error
			ssc, bformat, errs2 = mp3.Decode(fsteam)
			if errs2 != nil {
				report(errs2)
			}
			speaker.Clear()
			time.Sleep(time.Second / 2)
			speaker.Init(bformat.SampleRate, bformat.SampleRate.N(time.Second/10))
			ap = nil
			ap = newAudioPanel(bformat.SampleRate, ssc)
			nowplaytitle = listarray[nowdispindex].filename
			ap.play()
			return true, false
		}

		if event.Key() != tcell.KeyRune {
			return false, false
		}

		//处理按键
		switch unicode.ToLower(event.Rune()) {
		case ' ':
			speaker.Lock()
			if ap != nil {
				ap.ctrl.Paused = !ap.ctrl.Paused
			}
			speaker.Unlock()
			return false, false
		case 'q', 'e':
			//播放时间
			speaker.Lock()
			newPos := ap.streamer.Position()
			if event.Rune() == 'q' {
				newPos -= ap.sampleRate.N(time.Second * 5)
			}
			if event.Rune() == 'e' {
				newPos += ap.sampleRate.N(time.Second * 5)
			}
			if newPos < 0 {
				newPos = 0
			}
			if newPos >= ap.streamer.Len() {
				newPos = ap.streamer.Len() - 1
			}
			if err := ap.streamer.Seek(newPos); err != nil {
				report(err)
			}
			speaker.Unlock()
			return true, false

		case 'w':
			//向上选择
			if nowdispindex-1 > -1 {
				nowdispindex--
			}
			return true, false
		case 's':
			//向下选择
			if nowdispindex+1 < nowdispmaxindex {
				nowdispindex++
			}
			return true, false
		case 'a':
			//减速播放
			speaker.Lock()
			if ap != nil {
				ap.resampler.SetRatio(ap.resampler.Ratio() * 15 / 16)
			}
			speaker.Unlock()
			return true, false

		case 'd':
			//加速播放
			speaker.Lock()
			if ap != nil {
				ap.resampler.SetRatio(ap.resampler.Ratio() * 16 / 15)
			}
			speaker.Unlock()
			return true, false
		case 'r':
			//增大音量
			speaker.Lock()
			if ap != nil {
				ap.volume.Volume += 0.1
			}
			speaker.Unlock()
			return true, false
		case 'f':
			//减小音量
			speaker.Lock()
			if ap != nil {
				ap.volume.Volume -= 0.1
			}
			speaker.Unlock()
			return true, false
		default:
			//未知键写在提示
			tip = string(unicode.ToLower(event.Rune()))
			return false, false
		}
	}
	return false, false
}

// 主函数
func main() {
	//扫描指定目录
	fmt.Println("scan start")
	files, err := ioutil.ReadDir(filepath)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if !file.IsDir() {
			if path.Ext(file.Name()) == ".mp3" {
				listarray = append(listarray, listrow{name: file.Name(), filename: file.Name()})
			}
		}
	}

	nowdispmaxindex = len(listarray)

	fmt.Println("scan ok")

	screen, err := tcell.NewScreen()
	if err != nil {
		report(err)
	}
	err = screen.Init()
	if err != nil {
		report(err)
	}
	defer screen.Fini()

	//提前绘制一帧
	screen.Clear()
	TDrawCls(screen)
	screen.Show()

	seconds := time.Tick(time.Millisecond * 25)
	events := make(chan tcell.Event)
	go func() {
		for {
			events <- screen.PollEvent()
		}
	}()

loop:
	for {
		select {
		case event := <-events:
			change, exit := THandle(event)
			if exit {
				break loop
			}
			if change {
				//发生改变则重新绘制
				screen.Clear()
				TDrawCls(screen)
				screen.Show()
			}
		case <-seconds:
			if ap != nil && lrcEngine != nil {
				nowlrc = getNowLyric(ap.sampleRate.D(ap.streamer.Position()), lrcEngine)
			} else {
				nowlrc = "暂无歌词"
			}
			screen.Clear()
			TDrawCls(screen)
			screen.Show()
		}
	}
}

// 错误处理
func report(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// 格式化时间
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func ReadFile(path string) string {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("read fail", err)
		return ""
	}
	return string(f)
}

func GetFileName(filePath string) string {
	return strings.TrimSuffix(path.Base(filePath), path.Ext(filePath))
}
