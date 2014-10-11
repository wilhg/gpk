package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

/*
TODO List
Monitor - State & Files
to be a service, use C/S mode to connect
*/
type MyProcess struct {
	cmd          *exec.Cmd
	restartTimes int

	Id        int
	Tag       string
	StartTime time.Time
}

func NewProcess(tag string, id int, args []string) *MyProcess {
	return &MyProcess{
		cmd:       exec.Command(args[0], args[1:]...),
		Tag:       tag,
		Id:        id,
		StartTime: time.Now()}
}
func (mp *MyProcess) start() error {
	return mp.cmd.Start()
}
func (mp *MyProcess) wait() error {
	return mp.cmd.Wait()
}
func (mp *MyProcess) release() error {
	return mp.cmd.Process.Release()
}
func (mp *MyProcess) kill() error {
	return mp.cmd.Process.Kill()
}
func (mp *MyProcess) getPid() int {
	return mp.cmd.Process.Pid
}
func (mp *MyProcess) getState() *os.ProcessState {
	return mp.cmd.ProcessState
}
func (mp *MyProcess) recovery() error {
	mp.cmd = exec.Command(mp.cmd.Path, mp.cmd.Args[1:]...)
	mp.restartTimes++
	return mp.start()
}
func (mp *MyProcess) stop() error {
	if err := mp.kill(); err != nil && err.Error() != "no such process" {
		return err
	}
	return mp.release()
}
func (mp *MyProcess) restart() error {
	if err := mp.stop(); err != nil {
		return err
	}
	return mp.recovery()
}

type exitedProc struct {
	p        *MyProcess
	exitInfo error
}
type BigBrother struct {
	count int
	procs []*MyProcess
	eProc chan exitedProc
}

func NewBigBrother() *BigBrother {
	return &BigBrother{eProc: make(chan exitedProc, 100)}
}
func (bb *BigBrother) watch() {
	for {
		runtime.Gosched()
		select {
		case ep := <-bb.eProc:
			fmt.Println(ep.exitInfo)
			fmt.Println(ep.p.cmd.Args)
			ep.p.recovery()
			if ep.exitInfo != nil {

				// fmt.Println(ep.p.cmd.Args)
				return
			}
			go func() {
				bb.eProc <- exitedProc{ep.p, ep.p.wait()}
			}()
		}
	}
}
func (bb *BigBrother) register(mp *MyProcess) *BigBrother {
	mp.start()
	go func() {
		bb.eProc <- exitedProc{mp, mp.wait()}
	}()
	bb.procs = append(bb.procs, mp)
	return bb
}
func (bb *BigBrother) FindById(i int) (error, *MyProcess) {
	for _, p := range bb.procs {
		if p.Id == i {
			return nil, p
		}
	}
	return fmt.Errorf("No such ID"), &MyProcess{}
}
func (bb *BigBrother) FindByTag(tag string) (error, []*MyProcess) {
	var ps []*MyProcess
	for _, p := range bb.procs {
		if p.Tag == tag {
			ps = append(ps, p)
		}
	}
	if len(ps) > 0 {
		return nil, ps
	}
	return fmt.Errorf("No such Tag"), ps
}
func main() {
	p := NewProcess("sleep", 1, []string{"ps", "a"})
	bb := NewBigBrother()
	bb.register(p)
	bb.watch()
}
