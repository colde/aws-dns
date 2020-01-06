package main

import (
	"flag"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/miekg/dns"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

var hosts = make(map[string][]net.IP)
var ec2Svc *ec2.EC2

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	requestedHost := strings.Split(r.Question[0].Name, ".")[0]

	println("Handling request for", requestedHost)

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.RecursionAvailable = false
	for _, hostIP := range hosts[requestedHost] {
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 10},
			A:   hostIP,
		})
	}
	w.WriteMsg(m)
}

func getInstances() {
	newHosts := make(map[string][]net.IP)
	result, err := ec2Svc.DescribeInstances(nil)
	if err != nil {
		panic("Unable to get instances")
	} else {
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				for _, tag := range instance.Tags {
					if *tag.Key == "Role" {
						for _, networkInterface := range instance.NetworkInterfaces {
							newHosts[*tag.Value] = append(newHosts[*tag.Value], net.ParseIP(*networkInterface.PrivateIpAddress))
						}
					}
				}
			}
		}
	}
	hosts = newHosts
}

func main() {
	port := flag.Int("port", 8053, "port to run on")
	region := flag.String("region", "eu-north-1", "aws region to use")
	flag.Parse()

	hosts["test"] = append(hosts["test"], net.IPv4(1, 2, 3, 4))

	dns.HandleFunc("services.internal.", handleRequest)

	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String(*region)},
	)

	ec2Svc = ec2.New(sess)

	getInstances()

	go func() {
		srv := &dns.Server{Addr: ":" + strconv.Itoa(*port), Net: "udp"}
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Failed to set udp listener %s\n", err.Error())
		}
	}()

	go func() {
		srv := &dns.Server{Addr: ":" + strconv.Itoa(*port), Net: "tcp"}
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Failed to set tcp listener %s\n", err.Error())
		}
	}()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Fatalf("Signal (%v) received, stopping\n", s)
}
