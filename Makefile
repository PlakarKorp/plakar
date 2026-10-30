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

# Build the web UI into ui/web/build.
ui:
	cd ui/web/src && pnpm install --frozen-lockfile && pnpm --filter oss run build
	find ui/web/build -mindepth 1 -maxdepth 1 ! -name .gitkeep -exec rm -rf {} +
	cp -R ui/web/src/apps/oss/dist/. ui/web/build/
	@# Stamp the sources it was built from, which the release workflow checks
	@# against the tag. Uncommitted changes make the stamp match nothing.
	@if [ -n "$$(git status --porcelain ui/web/src)" ]; then echo dirty; \
	else git rev-parse HEAD:ui/web/src; fi > ui/web/build/.src-tree

# Build the UI and stage it for a release commit. Only release branches
# commit ui/web/build: main ignores it, hence the -f.
release-ui: ui
	git add -f ui/web/build

# Put ui/web/build back to what is checked out: drop what make ui wrote, and
# on a release branch restore the committed build.
ui-clean:
	git restore --staged --worktree ui/web/build
	git clean -fdX ui/web/build

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

junit:
	${GO} test -v -p 4 -coverprofile=coverage.out -covermode=atomic -timeout 2m -json ./... \
		| ${GO} tool go-junit-report -parser gojson -set-exit-code > junit.xml

# coverage runs the test suite and reports total statement coverage with the
# testing/ support packages (mocks, fixtures) excluded. Go has no package-level
# coverage exclude, so we filter the profile after the fact: the `mode:` header
# line is kept, and every block belonging to a testing/ package is dropped
# before `go tool cover` totals it.
COVERPROFILE = coverage.out
coverage:
	${GO} test -covermode=atomic -coverprofile=${COVERPROFILE} ./...
	@grep -v '/plakar/testing/' ${COVERPROFILE} > ${COVERPROFILE}.tmp && mv ${COVERPROFILE}.tmp ${COVERPROFILE}
	@${GO} tool cover -func=${COVERPROFILE} | tail -1

.PHONY: all plakar ui release-ui ui-clean install check test coverage junit
