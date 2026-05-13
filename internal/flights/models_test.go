package flights_test

import (
	"testing"

	"github.com/sperax/flight-price-service/internal/flights"
)

func TestValidate_AllFieldsValid(t *testing.T) {
	req := flights.FlightSearchRequest{
		Origin:      "GRU",
		Destination: "JFK",
		Date:        "2026-06-10",
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_MissingOrigin(t *testing.T) {
	req := flights.FlightSearchRequest{
		Destination: "JFK",
		Date:        "2026-06-10",
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for missing origin")
	}
	if err.Error() != "origin is required" {
		t.Fatalf("unexpected message: %s", err.Error())
	}
}

func TestValidate_MissingDestination(t *testing.T) {
	req := flights.FlightSearchRequest{
		Origin: "GRU",
		Date:   "2026-06-10",
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for missing destination")
	}
	if err.Error() != "destination is required" {
		t.Fatalf("unexpected message: %s", err.Error())
	}
}

func TestValidate_MissingDate(t *testing.T) {
	req := flights.FlightSearchRequest{
		Origin:      "GRU",
		Destination: "JFK",
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for missing date")
	}
	if err.Error() != "date is required" {
		t.Fatalf("unexpected message: %s", err.Error())
	}
}

func TestValidate_InvalidDateFormat(t *testing.T) {
	cases := []string{
		"10/06/2026",
		"2026/06/10",
		"June 10 2026",
		"invalid-date",
		"20260610",
	}

	for _, d := range cases {
		req := flights.FlightSearchRequest{
			Origin:      "GRU",
			Destination: "JFK",
			Date:        d,
		}
		err := req.Validate()
		if err == nil {
			t.Fatalf("expected error for date %q, got nil", d)
		}
		if err.Error() != "date must use YYYY-MM-DD format" {
			t.Fatalf("unexpected message for date %q: %s", d, err.Error())
		}
	}
}

func TestValidate_ValidDateFormats(t *testing.T) {
	cases := []string{
		"2026-01-01",
		"2026-12-31",
		"2026-06-10",
	}

	for _, d := range cases {
		req := flights.FlightSearchRequest{
			Origin:      "GRU",
			Destination: "JFK",
			Date:        d,
		}
		if err := req.Validate(); err != nil {
			t.Fatalf("expected valid date %q to pass, got: %v", d, err)
		}
	}
}
