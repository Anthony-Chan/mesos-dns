package main

import (
	"flag"
	"github.com/mesosphere/mesos-dns/logging"
	"github.com/mesosphere/mesos-dns/records"
	"github.com/mesosphere/mesos-dns/resolver"
	"github.com/miekg/dns"
	"sync"
	"time"
)

// init provies daemon flags
func init() {
	flag.BoolVar(&logging.VerboseFlag, "v", false, "increase the verbosity level")
}

func main() {
	var wg sync.WaitGroup
	var resolver resolver.Resolver

	flag.Parse()
	logging.SetupLogs()

	resolver.Config = records.SetConfig()

	// reload the first time
	resolver.Reload()
	ticker := time.NewTicker(time.Second * time.Duration(resolver.Config.Refresh))
	go func() {
		for _ = range ticker.C {
			resolver.Reload()
		}
	}()

	// handle for everything in this domain...
	dns.HandleFunc(resolver.Config.Domain+".", panicRecover(resolver.HandleMesos))
	dns.HandleFunc(".", panicRecover(resolver.HandleNonMesos))

	go resolver.Serve("tcp")
	go resolver.Serve("udp")

	wg.Add(1)
	wg.Wait()
}

func panicRecover(f func(w dns.ResponseWriter, r *dns.Msg)) func(w dns.ResponseWriter, r *dns.Msg) {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		defer func() {
			if rec := recover(); rec != nil {
				m := new(dns.Msg)
				m.SetReply(r)
				m.SetRcode(r, 2)
				_ = w.WriteMsg(m)
				logging.Error.Println(rec)
			}
		}()
		f(w, r)
	}
}
