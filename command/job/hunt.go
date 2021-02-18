package job

import (
	"fmt"
	"github.com/hashicorp/nomad/api"
	"strings"
	"time"
)

func Hunt() error {
	// create Nomad API client
	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	jobs, _, err := nomadClient.Jobs().List(nil)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if job.Type == "service" {
			jAllocations, _, _ := nomadClient.Jobs().Allocations(job.ID, true, nil)
			damn := false
			for _, allocation := range jAllocations {
				if allocation.JobVersion != jAllocations[0].JobVersion && allocation.ClientStatus == "running" {
					damn = true
					break
				}
			}
			if damn {
				fmt.Println(job.ID)
				for _, allocation := range jAllocations {

					fmt.Printf("%s | Version %v | Desired %s | Actual %s - %s | Create Time: %s\n", strings.Split(allocation.ID, "-")[0], allocation.JobVersion, allocation.DesiredStatus, allocation.ClientStatus, allocation.ClientDescription, prettyTimeDiff(time.Unix(0, allocation.CreateTime), time.Now()))
				}
				fmt.Println()
			}
		}
	}
	return nil
}


// prettyTimeDiff prints a human readable time difference.
// It uses abbreviated forms for each period - s for seconds, m for minutes, h for hours,
// d for days, mo for months, and y for years. Time difference is rounded to the nearest second,
// and the top two least granular periods are returned. For example, if the time difference
// is 10 months, 12 days, 3 hours and 2 seconds, the string "10mo12d" is returned. Zero values return the empty string
func prettyTimeDiff(first, second time.Time) string {
	// handle zero values
	if first.IsZero() || first.UnixNano() == 0 {
		return ""
	}
	// round to the nearest second
	first = first.Round(time.Second)
	second = second.Round(time.Second)

	// calculate time difference in seconds
	var d time.Duration
	messageSuffix := "ago"
	if second.Equal(first) || second.After(first) {
		d = second.Sub(first)
	} else {
		d = first.Sub(second)
		messageSuffix = "from now"
	}

	u := uint64(d.Seconds())

	var buf [32]byte
	w := len(buf)
	secs := u % 60

	// track indexes of various periods
	var indexes []int

	if secs > 0 {
		w--
		buf[w] = 's'
		// u is now seconds
		w = fmtInt(buf[:w], secs)
		indexes = append(indexes, w)
	}
	u /= 60
	// u is now minutes
	if u > 0 {
		mins := u % 60
		if mins > 0 {
			w--
			buf[w] = 'm'
			w = fmtInt(buf[:w], mins)
			indexes = append(indexes, w)
		}
		u /= 60
		// u is now hours
		if u > 0 {
			hrs := u % 24
			if hrs > 0 {
				w--
				buf[w] = 'h'
				w = fmtInt(buf[:w], hrs)
				indexes = append(indexes, w)
			}
			u /= 24
		}
		// u is now days
		if u > 0 {
			days := u % 30
			if days > 0 {
				w--
				buf[w] = 'd'
				w = fmtInt(buf[:w], days)
				indexes = append(indexes, w)
			}
			u /= 30
		}
		// u is now months
		if u > 0 {
			months := u % 12
			if months > 0 {
				w--
				buf[w] = 'o'
				w--
				buf[w] = 'm'
				w = fmtInt(buf[:w], months)
				indexes = append(indexes, w)
			}
			u /= 12
		}
		// u is now years
		if u > 0 {
			w--
			buf[w] = 'y'
			w = fmtInt(buf[:w], u)
			indexes = append(indexes, w)
		}
	}
	start := w
	end := len(buf)

	// truncate to the first two periods
	numPeriods := len(indexes)
	if numPeriods > 2 {
		end = indexes[numPeriods-3]
	}
	if start == end { //edge case when time difference is less than a second
		return "0s " + messageSuffix
	} else {
		return string(buf[start:end]) + " " + messageSuffix
	}

}

// fmtInt formats v into the tail of buf.
// It returns the index where the output begins.
func fmtInt(buf []byte, v uint64) int {
	w := len(buf)
	for v > 0 {
		w--
		buf[w] = byte(v%10) + '0'
		v /= 10
	}
	return w
}
