package server

import (
	"os"
	"strconv"

	"github.com/libp2p/zeroconf/v2"
	"github.com/sirupsen/logrus"
)

// mDNSService is the DNS-SD service type the desktop client browses for.
const mDNSService = "_nasai._tcp"

// startMDNSAdvertise announces the API over mDNS/DNS-SD so the desktop client
// can discover it on the LAN (browse "_nasai._tcp"). It returns a stop function
// that unregisters the service on shutdown.
//
// Disabled when MDNS_DISABLE is set. Advertising is best-effort: a failure is
// logged as a warning and never blocks server startup. Note that mDNS is
// link-local multicast — from a bridged Docker container it will not reach the
// host LAN; run the API on the host (or host networking) for real discovery.
func startMDNSAdvertise(port string, logger logrus.FieldLogger) (func(), error) {
	if os.Getenv("MDNS_DISABLE") != "" {
		return func() {}, nil
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return func() {}, err
	}

	instance, _ := os.Hostname()
	if instance == "" {
		instance = "nas-ai"
	}

	srv, err := zeroconf.Register(
		instance,
		mDNSService,
		"local.",
		p,
		[]string{"app=nas-ai", "tls=1"},
		nil, // alle Interfaces
	)
	if err != nil {
		return func() {}, err
	}

	logger.WithFields(logrus.Fields{
		"service":  mDNSService,
		"instance": instance,
		"port":     p,
	}).Info("mDNS: advertising NAS.AI service")

	return func() { srv.Shutdown() }, nil
}
