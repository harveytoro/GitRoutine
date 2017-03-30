package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"bytes"

	"github.com/jroimartin/gocui"
)

const repositoryView string = "repositoryView"
const summaryView string = "summaryView"
const terminalView string = "terminalView"
const repositoryNameView string = "repositoryNameView"
const branchNameView string = "branchNameView"
const logNameView string = "logNameView"

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
	g.SetKeybinding(terminalView, gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, _ *gocui.View) error {
		v, err := g.View(terminalView)
		if err != nil {
			return err
		}

		inputTerminal := v.Buffer()
		v.Clear()
		fmt.Fprintln(v, "git > ")
		v.SetCursor(6, 0)
		p, err := g.View(summaryView)
		if err != nil {
			return err
		}

		err = executeCommand(p, strings.TrimSpace(inputTerminal))

		if err != nil {
			return err
		}

		return nil
	})

	g.SetKeybinding(repositoryView, gocui.MouseLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		p, err := g.View(summaryView)
		repoNameView, err := g.View(repositoryNameView)
		branchView, err := g.View(branchNameView)
		logView, err := g.View(logNameView)
		if err != nil {
			return err
		}

		p.Clear()
		p.SetCursor(0, 0)

		_, cy := v.Cursor()
		line, err := v.Line(cy)
		for _, repos := range uiMgr.Repositories {
			if repos.Name == line {
				os.Chdir(repos.Path)
				repoNameView.Clear()
				fmt.Fprintln(repoNameView, "Repository: "+repos.Name)
				branchView.Clear()
				logView.Clear()
				logView.SetOrigin(0, 0)
				fmt.Fprint(branchView, "Current branch: ")
				executeCommand(branchView, "git > rev-parse --abbrev-ref HEAD")
				executeCommand(logView, "git > -c color.ui=always log --all --decorate --oneline --graph")
			}
		}

		return nil
	})

	g.SetKeybinding("", gocui.MouseLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		p, err := g.View(terminalView)
		if err != nil {
			return err
		}
		p.Clear()
		p.SetCursor(6, 0)
		fmt.Fprintln(p, "git > ")

		return nil
	})

	g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		v, _ = g.View(logNameView)
		if err != nil {
			return err
		}
		ox, oy := v.Origin()
		v.SetOrigin(ox, oy+1)

		return nil
	})

	g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		v, _ = g.View(logNameView)
		if err != nil {
			return err
		}
		ox, oy := v.Origin()
		v.SetOrigin(ox, oy-1)

		return nil
	})

	g.SetKeybinding(terminalView, gocui.KeyBackspace2, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		x, _ := v.Cursor()
		if x > 6 {
			v.EditDelete(true)
		}

		return nil
	})

	g.SetKeybinding("", gocui.MouseWheelDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		return nil
	})
	g.SetKeybinding("", gocui.MouseWheelUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

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

	if v, err := g.SetView(repositoryView, 1, 0, (maxX/2)/2, maxY-2); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Repositories"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen

		for _, repo := range mgr.Repositories {
			fmt.Fprintln(v, repo.Name)
		}
	}

	if v, err := g.SetView(summaryView, ((maxX/2)/2)+1, 0, maxX-1, maxY/2); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Summary"
		v.Autoscroll = true

	}

	if v, err := g.SetView(repositoryNameView, ((maxX/2)/2)+1, (maxY/2)+1, (maxX+((maxX/2)/2)+1)/2, (maxY/2)+3); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintln(v, "Repository:")
	}

	if v, err := g.SetView(branchNameView, (maxX+((maxX/2)/2)+1)/2+1, (maxY/2)+1, maxX-1, (maxY/2)+3); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintln(v, "Current branch:")
	}

	if v, err := g.SetView(logNameView, ((maxX/2)/2)+1, (maxY/2)+4, maxX-1, (maxY - 5)); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Log"
		v.Wrap = true
	}

	if v, err := g.SetView(terminalView, ((maxX/2)/2)+1, maxY-4, maxX-1, maxY-2); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		v.Editable = true
		v.Title = "Terminal"
		fmt.Fprintln(v, "git > ")
		v.SetCursor(6, 0)
	}

	g.SetCurrentView(terminalView)

	return nil
}

func executeCommand(view *gocui.View, command string) error {

	var (
		output    bytes.Buffer
		errOutput bytes.Buffer
		err       error
	)

	s := strings.Split(command, " ")

	cmd := "git"

	execCmd := exec.Command(cmd, s[2:]...)
	execCmd.Stderr = &errOutput
	execCmd.Stdout = &output

	err = execCmd.Run()
	if err != nil {
		log.Panicln(err)
		return nil
	}

	fmt.Fprintln(view, output.String())

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
