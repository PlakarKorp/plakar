GO =		go

DESTDIR =
PREFIX =	/usr/local
BINDIR =	${PREFIX}/bin
MANDIR =	${PREFIX}/man

INSTALL =	install
INSTALL_PROGRAM=${INSTALL} -m 0555
INSTALL_MAN =	${INSTALL} -m 0444

all: plakar

plakar:
	${GO} build -v .

install:
	mkdir -p ${DESTDIR}${BINDIR}
	mkdir -p ${DESTDIR}${MANDIR}/man1
	mkdir -p ${DESTDIR}${MANDIR}/man7
	${INSTALL_PROGRAM} plakar ${DESTDIR}${BINDIR}
	${INSTALL_MAN} *.1 ${DESTDIR}${MANDIR}/man1
	${INSTALL_MAN} *.7 ${DESTDIR}${MANDIR}/man7

check: test
test:
	${GO} test ./...

junit:
	${GO} test -v -p 4 -coverprofile=coverage.out -covermode=atomic -timeout 2m -json ./... \
		| ${GO} tool go-junit-report -parser gojson -set-exit-code > junit.xml

# coverage runs the test suite and reports total statement coverage with the
# testing/ support packages (mocks, fixtures) excluded, matching the Codecov
# `ignore` config. Go has no package-level coverage exclude, so we filter the
# profile after the fact: the `mode:` header line is kept, and every block
# belonging to a testing/ package is dropped before `go tool cover` totals it.
COVERPROFILE = coverage.out
coverage:
	${GO} test -covermode=atomic -coverprofile=${COVERPROFILE} ./...
	@grep -v '/plakar/testing/' ${COVERPROFILE} > ${COVERPROFILE}.tmp && mv ${COVERPROFILE}.tmp ${COVERPROFILE}
	@${GO} tool cover -func=${COVERPROFILE} | tail -1

.PHONY: all plakar install check test coverage junit
