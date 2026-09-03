package google

import (
	"net/url"
	"testing"
)

const sharedDriveID = "0AKbllKZ8knmBUk9PVA"

func TestApplySupportsAllDrives(t *testing.T) {
	t.Parallel()

	q := url.Values{}
	applySupportsAllDrives(q)
	if q.Get("supportsAllDrives") != "true" {
		t.Errorf("expected supportsAllDrives=true, got %q", q.Get("supportsAllDrives"))
	}
}

func TestApplyDriveListScope_AllDrives(t *testing.T) {
	t.Parallel()

	q := url.Values{}
	applyDriveListScope(q, "")
	assertAllDrivesList(t, q)
	if q.Get("driveId") != "" {
		t.Errorf("expected no driveId when driveID is empty, got %q", q.Get("driveId"))
	}
}

func TestApplyDriveListScope_SpecificDrive(t *testing.T) {
	t.Parallel()

	q := url.Values{}
	applyDriveListScope(q, sharedDriveID)
	assertDriveCorpus(t, q, sharedDriveID)
}

func TestIsValidDriveID_SharedDrive(t *testing.T) {
	t.Parallel()

	if !isValidDriveID(sharedDriveID) {
		t.Errorf("expected Shared Drive ID %q to be valid", sharedDriveID)
	}
}

func assertSupportsAllDrives(t *testing.T, q url.Values) {
	t.Helper()
	if q.Get("supportsAllDrives") != "true" {
		t.Errorf("expected supportsAllDrives=true, got %q", q.Get("supportsAllDrives"))
	}
}

func assertAllDrivesList(t *testing.T, q url.Values) {
	t.Helper()
	assertSupportsAllDrives(t, q)
	if q.Get("includeItemsFromAllDrives") != "true" {
		t.Errorf("expected includeItemsFromAllDrives=true, got %q", q.Get("includeItemsFromAllDrives"))
	}
	if q.Get("corpora") != "allDrives" {
		t.Errorf("expected corpora=allDrives, got %q", q.Get("corpora"))
	}
}

func assertDriveCorpus(t *testing.T, q url.Values, driveID string) {
	t.Helper()
	assertSupportsAllDrives(t, q)
	if q.Get("includeItemsFromAllDrives") != "true" {
		t.Errorf("expected includeItemsFromAllDrives=true, got %q", q.Get("includeItemsFromAllDrives"))
	}
	if q.Get("corpora") != "drive" {
		t.Errorf("expected corpora=drive, got %q", q.Get("corpora"))
	}
	if q.Get("driveId") != driveID {
		t.Errorf("expected driveId=%q, got %q", driveID, q.Get("driveId"))
	}
}
