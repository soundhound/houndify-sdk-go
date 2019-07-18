## v0.3.4 2019-07-17
Features:
* Pass the SafeToStopAudio flag recieved from the server with the PartialTranscript (See
  the streaming part of the example)
* Allow for adding context to the http.Request struct prior to the request being made
* Allow for adding extra arbitrary headers to the request

Changes:
* Updated example to demonstrate streaming audio & using SafeToStopAudio flag

Bugfixes:
* Fixed bug that if RequestInfo isn't provided a panic occurs, an empty map is used
  instead now

## v0.3.3 2019-07-09
Bugfixes:
* Changed the language headers to match expected format

## v0.3.2 2019-06-19
Changes:
* Updated the example audio files provided in the test_audio directory   

## v0.3.1 2019-06-06
Features:
*  Allow sending Request-Info in request body instead of in header
*  Started request test suite

Changes:
*  Refactor request generating code
*  Run tests in travis
*  Added error for empty server responses

New Dependencies:
* github.com/google/go-cmp
* github.com/pkg/errors
* gotest.tools

