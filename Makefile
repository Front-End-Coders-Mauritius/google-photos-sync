add:
	~/go/bin/timeliner add-account google_photos/front.end.coders.mu@gmail.com

init:
	~/go/bin/timeliner get-all google_photos/front.end.coders.mu@gmail.com

reauth:
	~/go/bin/timeliner reauth google_photos/front.end.coders.mu@gmail.com

sync:
	~/go/bin/timeliner get-latest google_photos/front.end.coders.mu@gmail.com

json:
	go run .
