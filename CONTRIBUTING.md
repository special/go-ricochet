# How To Contribute to GoRicochet

This document highlights some useful tips for contributing to this project. Feel
free to submit pull requests to update this document as needed.

# Requesting a New Feature

You can request a new feature by submitting an issue to the [Github Repository](https://github.com/s-rah/go-ricochet).

# Writing New Code

So you want to dig in? Awesome! Here are a few steps to consider:

## 1. Before working on a Change

First, check to see if your proposed change is already in active development. This
can be done by searching issues and pull requests on Github.

If no issue exists for your change then please open one. You **do not** need to wait
for the maintainers or the community to discuss your change before starting work but, for
larger changes, we suggest attaching some design notes and requesting feedback from the 
maintainers to avoid the change being rejected after all that hard work!

## 2. Make the Change

Crack open the editor of your choice and start programming. As you go along ensure
to run `go test github.com/s-rah/go-ricochet` to ensure that everything is working!

Please write tests for any new functionality. As a rule, aim for >80% code coverage. You
can check coverage `with go test --cover github.com/s-rah/go-ricochet`

## 3. Before Submitting a Pull Request

Format your code (the path might be slightly different):

* `gofmt -l=true -s -w src/github.com/s-rah/go-ricochet/`

Run the following commands, and address any issues which arise:

* `go vet github.com/s-rah/go-ricochet/...`
* `golint github.com/s-rah/go-ricochet`
* `golint github.com/s-rah/go-ricochet/utils`

## 4. Code Review

Once you submit the pull request it will be reviewed by one of the maintainers 
of the project. At this point there are 3 possible paths:

1. Your change is accepted!
2. Your change is acknowledged as necessary, but requires some rework before being accepted.
3. Your change is rejected - this can happen for a number of reasons. To minimize the chances of this happening, please see step #1
