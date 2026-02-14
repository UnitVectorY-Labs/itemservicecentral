This is a Go application that provides a configuration driven API that is delivered via a Docker container.

This application minimizes the user of dependencies preferring the standard Go library where possible.  The main.go file is the entry point and the rest of the code is organized under the internal/ directory. Any external files needed by the application such as a database schema or migration files are to be included in the compiled binary using the embed package.

While the bulk of the configuration lives in a configuration file and is referenced at runtime, the actual configuration passed into this application can be accomplished using well documented environment variables that each have a corresponding command line flag.  This allows for the application to be configured in a variety of ways and makes it easy to use in a containerized environment.

The documentation for this application lives in the docs/ directory and the root README.md file is minimal.

## Testing

When testing, use a docker container running the `postgres:18` image.
