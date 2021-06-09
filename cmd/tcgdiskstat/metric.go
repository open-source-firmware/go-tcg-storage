package main

import (
	"log"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

type metricCollector struct {
	m []prometheus.Metric
}

func (mc *metricCollector) Collect(c chan<- prometheus.Metric) {
	for _, m := range mc.m {
		c <- m
	}
}

func (mc *metricCollector) Describe(c chan<- *prometheus.Desc) {
}

func outputMetrics(state Devices) {
	var (
		mDriveInfo = prometheus.NewDesc(
			"tcg_storage_drive_info",
			"Info metric regarding the detected drives",
			[]string{"device", "model", "serial", "firmware", "protocol"}, nil,
		)
		mTCGSupported = prometheus.NewDesc(
			"tcg_storage_supported",
			"Boolean describing whether a drive supports any TCG storage standards",
			[]string{"device"}, nil,
		)
		mSSCSupported = prometheus.NewDesc(
			"tcg_storage_ssc_supported",
			"Boolean describing whether a particular SSC is supported by the drive or not",
			[]string{"device", "ssc"}, nil,
		)
		mLockingEnabled = prometheus.NewDesc(
			"tcg_storage_locking_enabled",
			"Boolean describing whether the drive is reporting range locking has been enabled",
			[]string{"device"}, nil,
		)
		mSIDAuthBlocked = prometheus.NewDesc(
			"tcg_storage_sid_authentication_blocked",
			"Boolean describing if the Block SID feature has made authentication to the drive currently impossible",
			[]string{"device"}, nil,
		)
		mDefaultSIDPIN = prometheus.NewDesc(
			"tcg_storage_default_sid_pin_detected",
			"Boolean describing if the Block SID feature reports the default SID PIN is in use",
			[]string{"device"}, nil,
		)
	)
	mc := &metricCollector{}
	for _, s := range state {
		mc.m = append(mc.m,
			prometheus.MustNewConstMetric(mDriveInfo, prometheus.GaugeValue, 1,
				s.Device, s.Identity.Model, s.Identity.SerialNumber, s.Identity.Firmware, s.Identity.Protocol))
		sup := float64(0)
		if s.Level0 != nil {
			sup = 1
		}
		mc.m = append(mc.m, prometheus.MustNewConstMetric(mTCGSupported, prometheus.GaugeValue, sup, s.Device))

		// This is how far we can make it without a successful Level0 discovery
		if s.Level0 == nil {
			continue
		}

		for _, ssc := range sscFeatures(s.Level0) {
			mc.m = append(mc.m,
				prometheus.MustNewConstMetric(mSSCSupported, prometheus.GaugeValue, 1,
					s.Device, ssc))
		}

		lockEn := float64(0)
		if l := s.Level0.Locking; l != nil {
			if l.LockingEnabled {
				lockEn = 1
			}
		}
		mc.m = append(mc.m, prometheus.MustNewConstMetric(mLockingEnabled, prometheus.GaugeValue, lockEn, s.Device))

		if b := s.Level0.BlockSID; b != nil {
			authBlock := float64(0)
			bDefaultSID := float64(0)
			if !b.SIDValueState {
				bDefaultSID = 1
			}
			if b.SIDAuthenticationBlockedState {
				authBlock = 1
			}
			// Metrics only visible if Block SID feature is supported
			mc.m = append(mc.m, prometheus.MustNewConstMetric(mSIDAuthBlocked, prometheus.GaugeValue, authBlock, s.Device))
			mc.m = append(mc.m, prometheus.MustNewConstMetric(mDefaultSIDPIN, prometheus.GaugeValue, bDefaultSID, s.Device))
		}
	}

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(mc)

	mfs, err := reg.Gather()
	if err != nil {
		log.Fatalf("Failed to gather metrics: %v", err)
	}
	for _, mf := range mfs {
		if _, err := expfmt.MetricFamilyToText(os.Stdout, mf); err != nil {
			log.Fatalf("Failed to serialize metrics: %v", err)
		}
	}
}
