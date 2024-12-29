main_package_path = ./cmd/server/
binary_name = cland
image_name = cland

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@gawk ' \
		/^#\s*=+/ { \
			in_header = 1; \
			next \
		} \
		in_header && /^# [A-Z]/ { \
			print "\n" substr($$0, 3); \
			in_header = 0; \
			next \
		} \
		/^##/ { \
			sub(/^##\s*/, ""); \
			comment = $$0; \
			getline name; \
			split(name, a, " "); \
			print "    " a[2] ":" comment; \
		}' \
	Makefile | column -t -s ':'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: no-dirty
no-dirty:
	@test -z "$(shell git status --porcelain)"

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## run quality control checks
.PHONY: audit
audit: test
	go mod tidy -diff
	go mod verify
	test -z "$(shell gofmt -l .)" 
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

## run all tests
.PHONY: test
test:
	go test -v -race -buildvcs ./...

## run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## tidy modfiles and format .go files
.PHONY: tidy
tidy:
	go mod tidy -v
	go fmt ./...

## build the application
.PHONY: build
build:
	go build -o=/tmp/bin/${binary_name} ${main_package_path}

## run the  application
.PHONY: run
run: build
	/tmp/bin/${binary_name}

## run the application with reloading on file changes
.PHONY: run/live
run/live:
	air --build.cmd "make build" --build.bin "/tmp/bin/${binary_name}" --build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "go, tpl, tmpl, html, css, scss, js, ts, sql, jpeg, jpg, gif, png, bmp, svg, webp, ico" \
		--misc.clean_on_exit "true"


# ==================================================================================== #
# DOCKER
# ==================================================================================== #

## build the Docker image
.PHONY: docker/build
docker/build:
	docker build -t $(image_name) .

## run the Docker container
.PHONY: docker/run
docker/run:
	docker run -it --rm -p 8080:8080 $(image_name)

# ==================================================================================== #
# Frontend
# ==================================================================================== #

## build the Tailwind css
.PHONY: tailwid
tailwind:
	@printf "\033[33mPlease uncomment the command to run tailwindcss and put in the correct paths to use this target\033[0m\n"
	# fd input.css | entr -r tailwindcss -w -i path/to/input.css -o path/to/output.css

## sort tailwind classes
.PHONY: rustywind
rustywind:
	fd -e templ -e html | entr -r rustywind --write .

## generate go files from templ files
.PHONY: templ
templ:
	find . -type f \( -name "*.templ" -or -name "*.css" -or -name "*.js" \) | entr -r bash -c 'TEMPL_EXPERIMENT=rawgo templ generate'

## run all frontend tasks in separate tmux panes
.PHONY: tmux-frontend
tmux-frontend:
	@tmux \
		split-window -h \; \
		send-keys 'make rustywind' C-m \; \
		split-window -v \; \
		send-keys 'make tailwind' C-m \; \
		select-pane -L \; \
		send-keys 'make templ' C-m \; \
