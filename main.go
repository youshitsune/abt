package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type Configuration struct {
	FQBN string
	Port string
}

var conf Configuration

const fileName = ".abt.json"

func (conf *Configuration) loadConfig() error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(conf)
	if err != nil {
		return err
	}

	return nil
}

func (conf *Configuration) saveConfig() error {
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(fmt.Sprintf("{\"FQBN\": \"%s\", \"Port\": \"%s\"}", conf.FQBN, conf.Port))
	if err != nil {
		return err
	}

	return nil
}

type item struct {
	name string
	fqbn string
}

func (i item) Title() string       { return i.name }
func (i item) Description() string { return i.fqbn }
func (i item) FilterValue() string { return i.name + i.fqbn }

type model struct {
	list   list.Model
	cursor int
	mode   int
}

func initalModel() model {
	items := make([]list.Item, 0)
	r := getBoardList()
	for _, v := range r {
		items = append(items, item{name: v[0], fqbn: v[1]})
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select the board"
	return model{
		list:   l,
		cursor: 0,
		mode:   0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			if m.mode == 0 {
				selectBoard(m.list.SelectedItem().(item).fqbn)
				ports := getPorts()
				var tmp []list.Item
				for _, v := range ports {
					tmp = append(tmp, item{name: v, fqbn: ""})
				}
				m.list.ResetFilter()
				cmds = append(cmds, m.list.SetItems(tmp))
				m.list.Title = "Select the port"
				m.mode = 1
			} else if m.mode == 1 {
				selectPort(m.list.SelectedItem().(item).name)
				if err := conf.saveConfig(); err != nil {
					fmt.Println(err)
				}
				return m, tea.Quit
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return m.list.View()
}

func getBoardList() [][]string {
	res := make([][]string, 0)
	r, _ := exec.Command("arduino-cli", []string{"board", "listall"}...).CombinedOutput()
	rows := strings.Split(string(r), "\n")
	for i := 1; i < len(rows); i++ {
		t := strings.Split(rows[i], " ")
		if len(t) > 1 {
			res = append(res, []string{strings.TrimSpace(strings.Join(t[:len(t)-2], " ")), strings.TrimSpace(t[len(t)-1])})
		}
	}
	return res
}

func getPorts() []string {
	res := make([]string, 0)
	r, _ := exec.Command("ls", "/dev").CombinedOutput()
	ports := strings.Split(string(r), "\n")
	for _, v := range ports {
		if strings.HasPrefix(v, "ttyUSB") || strings.HasPrefix(v, "ttyACM") {
			res = append(res, "/dev/"+v)
		}
	}

	return res
}

func selectBoard(fqbn string) {
	conf.FQBN = fqbn
}

func selectPort(port string) {
	conf.Port = port
}

func printHelp() {
	fmt.Println("Help")
}

func compile() {
	err := exec.Command("arduino-cli", []string{"compile", "-b", conf.FQBN}...).Run()
	if err != nil {
		fmt.Println(err)
	}
}

func upload() {
	err := exec.Command("arduino-cli", []string{"upload", "-b", conf.FQBN, "-p", conf.Port}...).Run()
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	conf.loadConfig()
	args := os.Args
	if len(args) < 2 {
		printHelp()
	} else {
		if args[1] == "c" || args[1] == "config" {
			p := tea.NewProgram(initalModel())
			p.Run()
		} else if args[1] == "a" || args[1] == "all" {
			compile()
			upload()
		} else if args[1] == "comp" || args[1] == "compile" {
			compile()
		} else if args[1] == "u" || args[1] == "upload" {
			upload()
		}
	}
}
