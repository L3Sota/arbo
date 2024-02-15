package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/L3Sota/arbo/arb"
	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/c"
	"github.com/L3Sota/arbo/g"
	"github.com/L3Sota/arbo/h"
	"github.com/L3Sota/arbo/k"
	"github.com/gregdel/pushover"
)

var (
	p *pushover.Pushover
	r *pushover.Recipient
)

func oneoff() {
	conf := config.Load()
	k.LoadClient(conf)
	h.LoadClient(conf)
	c.LoadClient(conf)
	g.LoadClient()

	if conf.PEnable {
		p = pushover.New(conf.PKey)
		r = pushover.NewRecipient(conf.PUser)
	}

	gatherBalances, msgs, err := arb.Book(true, conf)
	fmt.Println(gatherBalances)
	fmt.Println(msgs)
	fmt.Println(err)
}

func repeat() {
	deadline := time.NewTimer(59*time.Minute + 50*time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)

	conf := config.Load()
	k.LoadClient(conf)
	h.LoadClient(conf)
	c.LoadClient(conf)
	g.LoadClient()

	if conf.PEnable {
		p = pushover.New(conf.PKey)
		r = pushover.NewRecipient(conf.PUser)
	}

	var (
		gatherBalances = true
		msgs           []string
		err            error
	)
	for {
		fmt.Println("arb at", time.Now().String())
		gatherBalances, msgs, err = arb.Book(gatherBalances, conf)
		if err != nil {
			msg := fmt.Sprintf("[%v] arb ending due to error: %v", time.Now().String(), err.Error())
			fmt.Println(msg)
			if conf.PEnable {
				resp, err := p.SendMessage(&pushover.Message{
					Message: msg,
				}, r)
				if err != nil {
					fmt.Println("push err:", err.Error())
					return
				} else {
					fmt.Println("push ok:", resp.String())
				}
			}

			return
		}

		if conf.PEnable && len(msgs) > 0 {
			msg := strings.Join(msgs, "\n---\n")
			resp, err := p.SendMessage(&pushover.Message{
				Message: msg,
			}, r)
			if err != nil {
				fmt.Println("push err:", err.Error())
				return
			} else {
				fmt.Println("push ok:", resp.String())
			}
		}

		select {
		case t := <-deadline.C:
			fmt.Println("deadline reached, ending at", t.String())
			return
		case <-ticker.C:
			continue
		}
	}
}

func main() {
	repeat()
}
