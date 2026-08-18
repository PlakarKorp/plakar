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
	mkdir -p ${DESTDIR}${MANDIR}/man5
	mkdir -p ${DESTDIR}${MANDIR}/man7
	${INSTALL_PROGRAM} plakar ${DESTDIR}${BINDIR}
	find . -name \*.1 -exec ${INSTALL_MAN} {} ${DESTDIR}${MANDIR}/man1 \;
	find . -name \*.5 -exec ${INSTALL_MAN} {} ${DESTDIR}${MANDIR}/man5 \;
	find . -name \*.7 -exec ${INSTALL_MAN} {} ${DESTDIR}${MANDIR}/man7 \;

check: test
test:
	${GO} test ./...

.PHONY: all plakar install check test
