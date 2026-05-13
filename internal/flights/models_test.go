package flights_test

import (
	"errors"
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
	var valErr *flights.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "origin" {
		t.Fatalf("expected ValidationError with field=origin, got %T: %v", err, err)
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
	var valErr *flights.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "destination" {
		t.Fatalf("expected ValidationError with field=destination, got %T: %v", err, err)
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
	var valErr *flights.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "date" {
		t.Fatalf("expected ValidationError with field=date, got %T: %v", err, err)
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
		var valErr *flights.ValidationError
		if !errors.As(err, &valErr) || valErr.Field != "date" {
			t.Fatalf("expected ValidationError with field=date for %q, got %T: %v", d, err, err)
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
