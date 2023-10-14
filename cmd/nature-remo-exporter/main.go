package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kou64yama/nature-remo-exporter/internal/nature"
)

func main() {
	var (
		port int
		host string
	)

	flag.IntVar(&port, "p", 8080, "specify listening port")
	flag.StringVar(&host, "h", "127.0.0.1", "specify listening host")
	flag.Parse()

	var devices []nature.Device
	var appliances []nature.Appliance

	mu := http.NewServeMux()
	mu.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "# HELP natureremo_temperature Temperature\n")
		fmt.Fprintf(w, "# TYPE natureremo_temperature gauge\n")
		for _, d := range devices {
			fmt.Fprintf(
				w,
				"natureremo_temperature{name=%q,firmware_version=%q,mac_address=%q,serial_number=%q} %f\n",
				d.Name,
				d.FirmwareVersion,
				d.MacAddress,
				d.SerialNumber,
				d.NewestEvents.Temperature.Value,
			)
		}

		fmt.Fprintf(w, "# HELP natureremo_humidity Humidity\n")
		fmt.Fprintf(w, "# TYPE natureremo_humidity gauge\n")
		for _, d := range devices {
			fmt.Fprintf(
				w,
				"natureremo_humidity{name=%q,firmware_version=%q,mac_address=%q,serial_number=%q} %f\n",
				d.Name,
				d.FirmwareVersion,
				d.MacAddress,
				d.SerialNumber,
				d.NewestEvents.Humidity.Value,
			)
		}

		fmt.Fprintf(w, "# HELP natureremo_illumination Illumination\n")
		fmt.Fprintf(w, "# TYPE natureremo_illumination gauge\n")
		for _, d := range devices {
			fmt.Fprintf(
				w,
				"natureremo_illumination{name=%q,firmware_version=%q,mac_address=%q,serial_number=%q} %f\n",
				d.Name,
				d.FirmwareVersion,
				d.MacAddress,
				d.SerialNumber,
				d.NewestEvents.Illumination.Value,
			)
		}

		fmt.Fprintf(w, "# HELP natureremo_movement Movement\n")
		fmt.Fprintf(w, "# TYPE natureremo_movement gauge\n")
		for _, d := range devices {
			fmt.Fprintf(
				w,
				"natureremo_movement{name=%q,firmware_version=%q,mac_address=%q,serial_number=%q} %f\n",
				d.Name,
				d.FirmwareVersion,
				d.MacAddress,
				d.SerialNumber,
				d.NewestEvents.Movement.Value,
			)
		}
		
		fmt.Fprintf(w, "# HELP natureremo_appliance Appliance\n")
		fmt.Fprintf(w, "# TYPE natureremo_appliance gauge\n")
		for _, a := range appliances {
			for _, p := range a.Properties {
				val, _ := strconv.ParseInt(p.Value, 16, 64)
				fmt.Fprintf(
					w,
					"natureremo_appliance_%v{name=%q,firmware_version=%q,mac_address=%q,serial_number=%q} %d\n",
					p.EPC,
					a.Name,
					a.Device.FirmwareVersion,
					a.Device.MacAddress,
					a.Device.SerialNumber,
					val,
				)
			}
		}
	})

	var srv http.Server
	srv.Addr = host + ":" + strconv.Itoa(port)
	srv.Handler = mu

	idleConnsClosed := make(chan struct{})
	go func() {
		s := make(chan os.Signal, 1)
		signal.Notify(s, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
		<-s

		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	go func() {
		c := nature.NewClient(nature.AccessToken(os.Getenv("NATURE_ACCESS_TOKEN")))
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()

		if resultDevices, _ := c.GetDevices(); resultDevices != nil{
			devices = resultDevices
		}

		if resultAppliances, _ := c.GetAppliances(); resultAppliances != nil {
			appliances = resultAppliances
		}
		for {
			select {
			case <-idleConnsClosed:
				t.Stop()
				return
			case <-t.C:
				if resultDevices, _ := c.GetDevices(); resultDevices != nil {
					devices = resultDevices
				}

				if resultAppliances, _ := c.GetAppliances(); resultAppliances != nil {
					appliances = resultAppliances
				}
			}
		}
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
}
