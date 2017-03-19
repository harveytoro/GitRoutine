package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"bytes"

	"io/ioutil"

	"github.com/jroimartin/gocui"
)

type UIManager struct {
	configuration
}

func main() {

	config, _ := loadConfiguration()

	uiMgr := &UIManager{}
	uiMgr.configuration = config

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Cursor = true
	g.Mouse = true

	g.SetManagerFunc(uiMgr.layoutManager)
	g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, _ *gocui.View) error {
		v, err := g.View("terminal")
		if err != nil {
			return err
		}

		something := v.Buffer()

		p, err := g.View("summary")
		if err != nil {
			return err
		}

		//fmt.Fprintln(p, " source code "+config.SourceCodePath)

		dView, err := g.View("directory")
		if err != nil {
			log.Panicln(err)
		}
		//loadDirectory(dView, config.SourceCodePath)

		for _, value := range config.Repositories {
			fmt.Fprintln(dView, value.Name)
		}

		err = executeCommand(p, strings.TrimSpace(something))

		if err != nil {
			return err
		}

		v.Clear()
		v.SetCursor(0, 0)

		return nil
	})

	g.SetKeybinding("directory", gocui.MouseLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		p, err := g.View("summary")
		if err != nil {
			return err
		}

		_, cy := v.Cursor()
		line, err := v.Line(cy)
		var pathUsed string
		for _, repos := range uiMgr.Repositories {
			if repos.Name == line {
				os.Chdir(repos.Path)
				pathUsed = repos.Path
			}
		}

		fmt.Fprintln(p, pathUsed)

		return nil
	})

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func (mgr *UIManager) layoutManager(g *gocui.Gui) error {

	maxX, maxY := g.Size()

	if v, err := g.SetView("directory", 1, 0, (maxX/2)/2, maxY-2); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		v.Editable = true
		v.Title = "Repositories"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen

		for _, repo := range mgr.Repositories {
			fmt.Fprintln(v, repo.Name)
		}

	}

	if v, err := g.SetView("summary", ((maxX/2)/2)+1, 0, maxX-1, maxY/2); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		v.Editable = true
		v.Title = "Summary"
		v.Autoscroll = true

	}

	if v, err := g.SetView("sdfsdfsdf", ((maxX/2)/2)+1, (maxY/2)+1, ((maxX/2)/2)+10, (maxY/2)+3); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		v.Editable = true
		v.BgColor = gocui.ColorCyan
		fmt.Fprintln(v, "Status")
		//v.Title = "testing"
	}

	if v, err := g.SetView("terminal", ((maxX/2)/2)+1, (maxY/2)+4, maxX-1, maxY-2); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		v.Editable = true
		v.Title = "Terminal"
	}

	g.SetCurrentView("terminal")

	return nil
}

func executeCommand(view *gocui.View, command string) error {

	var (
		output    bytes.Buffer
		errOutput bytes.Buffer
		err       error
	)

	s := strings.Split(command, " ")

	cmd := s[0]
	fmt.Fprintln(view, "The command is: "+cmd)
	//args := []string{"--version"}
	//args := []string{}

	execCmd := exec.Command(cmd, s[1:]...)
	execCmd.Stderr = &errOutput
	execCmd.Stdout = &output

	err = execCmd.Run()
	if err != nil {
		fmt.Fprintln(view, fmt.Sprint(err)+" : "+errOutput.String())
		return nil
		//os.Exit(1)
	}

	fmt.Fprintln(view, "Executing "+command)
	fmt.Fprintln(view, output.String())
	//fmt.Fprintln(v, "Something happened")
	return nil
}

type repository struct {
	Name string
	Path string
}

type configuration struct {
	Repositories []repository
}

func loadConfiguration() (configuration, error) {

	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	configuration := configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		return configuration, err
	}

	return configuration, nil
}

func loadDirectory(view *gocui.View, dir string) {

	folders, err := ioutil.ReadDir(dir)

	if err != nil {
		log.Panicln(err)
	}
	for _, f := range folders {
		//parts := strings.Split(f.Name(), "\\")
		fmt.Fprintln(view, f.Name())
	}
}
