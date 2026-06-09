package protonmail

import "testing"

func TestSMTPTLSConfig_loopbackSkipsVerification(t *testing.T) {
	t.Parallel()
	for _, host := range []string{"127.0.0.1", "localhost", "::1"} {
		cfg := smtpTLSConfig(host)
		if !cfg.InsecureSkipVerify {
			t.Errorf("smtpTLSConfig(%q).InsecureSkipVerify = false, want true (Bridge serves a self-signed cert)", host)
		}
		if cfg.ServerName != host {
			t.Errorf("smtpTLSConfig(%q).ServerName = %q, want %q", host, cfg.ServerName, host)
		}
	}
}

func TestSMTPTLSConfig_remoteVerifies(t *testing.T) {
	t.Parallel()
	for _, host := range []string{"mail.example.com", "192.168.1.50"} {
		cfg := smtpTLSConfig(host)
		if cfg.InsecureSkipVerify {
			t.Errorf("smtpTLSConfig(%q).InsecureSkipVerify = true, want false (remote hosts must verify)", host)
		}
	}
}
