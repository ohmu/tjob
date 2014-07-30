default:
	@echo hmm...

dep:
	go get -u github.com/jessevdk/go-flags
	go get -u code.google.com/p/go.crypto/ssh

lint:
	golint -min_confidence=0.3 .|grep -v "should have comment" || true
	go vet

man:
	go build
	WRITE_MAN_PAGE=tjob.man ./tjob
	groff -Tascii -man tjob.man
