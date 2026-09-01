package web

import "testing"

func TestPreviewTicketsIssueLookup(t *testing.T) {
	s := newPreviewTickets()
	id, err := s.Issue(7, 42)
	if err != nil || id == "" {
		t.Fatalf("issue: %v %s", err, id)
	}
	uid, fileId, ok := s.Lookup(id)
	if !ok || uid != 7 || fileId != 42 {
		t.Fatalf("lookup got %d %d %v", uid, fileId, ok)
	}
	if _, _, ok = s.Lookup("nope"); ok {
		t.Fatal("unknown ticket should miss")
	}
	uid, fileId, ok = s.Lookup(id)
	if !ok || uid != 7 || fileId != 42 {
		t.Fatal("ticket should be reusable until expire")
	}
	// 登录 JWT 不能当预览票用
	if _, _, ok = s.Lookup("eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.e30.xx"); ok {
		t.Fatal("foreign jwt should miss")
	}
}

func TestFileMIME(t *testing.T) {
	if fileMIME("a.PNG") != "image/png" {
		t.Fatal(fileMIME("a.PNG"))
	}
	if fileMIME("v.mp4") != "video/mp4" {
		t.Fatal(fileMIME("v.mp4"))
	}
	if fileMIME("s.mp3") != "audio/mpeg" {
		t.Fatal(fileMIME("s.mp3"))
	}
	if fileMIME("s.m4a") != "audio/mp4" {
		t.Fatal(fileMIME("s.m4a"))
	}
}
