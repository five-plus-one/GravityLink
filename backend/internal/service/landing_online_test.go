package service

import "testing"

func TestIsOnlineNowDefaultAlwaysOnline(t *testing.T) {
	if !isOnlineNow(nil) {
		t.Fatal("nil schedule should be online")
	}
	empty := ""
	if !isOnlineNow(&empty) {
		t.Fatal("empty schedule should be online")
	}
	invalid := "not-json"
	if !isOnlineNow(&invalid) {
		t.Fatal("invalid JSON should fall back to online")
	}
}

func TestIsOnlineNowUsesWeekdaySlots(t *testing.T) {
	// 覆盖所有 7 天全天，必然在线
	all := `{"0":[["00:00","23:59"]],"1":[["00:00","23:59"]],"2":[["00:00","23:59"]],"3":[["00:00","23:59"]],"4":[["00:00","23:59"]],"5":[["00:00","23:59"]],"6":[["00:00","23:59"]]}`
	if !isOnlineNow(&all) {
		t.Fatal("all-day schedule should be online")
	}
	// 无任何时段
	none := `{"0":[]}`
	if isOnlineNow(&none) {
		t.Fatal("empty day list should be offline")
	}
}
