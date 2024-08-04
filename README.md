# QuizDrum
QuizDrum is a piece of software that allows you to host a website for conducting and participating in quizzes. Quizmasters can log on to your QuizDrum website with their Google account and create new quizzes, add and modify questions, and host a live quizzing session. Participants can log on to your QuizDrum website and answer and update questions and keep track of the scoreboard. The quizmaster can awards points to each answer, and control the flow of the quiz.

You can install QuizDrum on most operating systems, but it has only been tested on linux and windows machines. It is written in Go and JavaScript, and stores its data using SQLite. There are no pre-set limits on the number of participants.

*This is not an officially supported Google product.*

## Installation
The QuizDrum backend is written in the Go programming language, and uses SQLite for data storage. You should first [install Go](https://golang.org/dl/), if you don't already have it. Since the go version of SQLite requires `gcc` in the path during installation, and requires the variable `CGO_ENABLED=1` to be set in the environment, please see the [instructions on installing SQLite](https://github.com/mattn/go-sqlite3#installation) first, before proceeding further. Particularly on Windows, where having `gcc` in the path is uncommon, you may have to install the 64-bit `gcc` from a source like [tdm-gcc](https://jmeubank.github.io/tdm-gcc/).
1. Confirm you have installed Go and SQLite as described above
2. Clone this repository and `cd` to it
3. `go build`

In order to allow quizmasters to log in, you will need to set up a client ID so that they can log in with their Google Account. To do so, follow the steps in the [Create Authorization Credentials](https://developers.google.com/identity/protocols/oauth2/javascript-implicit-flow#creatingcred) section, and save the client id. You can then run quizdrum as
```
quizdrum --port=80 --oauth_client_id=<your-google-client-id>
```
QuizDrum has to be run from the source folder since it depends on the template files in the source tree. You could also [set it up as a service](https://medium.com/@benmorel/creating-a-linux-service-with-systemd-611b5c8b91d6). You can also run it on a different port and then set up [a reverse proxy to it](https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/).

## Use
Quizmasters should log in with their Google Account. Once logged in, they can create a new quiz from the homepage. On the quiz editing page, they can add, remove, and modify questions, as well as change the name and description of the quiz. Questions can of five types: short text answers, multiple choice, true or false, numeric answers with integers, and decimal numbers.

Once the quizmaster is satisfied with the quiz, they can go to the homepage and press the **Present** button to begin the quiz session. Participants can visit the quiz website and log in (either with their Google Account, or as a Guest). They need to click the **Participate** button next to the quiz to participate in, and then they can choose a screen name for that particular quiz.

As the quizmaster proceeds through the quiz, they can move back and forth over questions. They can stop accepting answers at any time, and awards points to the answers they receive. The page will update automatically with answers as they are submitted and updated. The quizmaster will have to click **Save Scores** to finalize awarding the points, at which point the scoreboard will update with the scores.

## Screenshots
![Screenshot of a quizmaster using QuizDrum to edit a quiz. A panel is visible with questions in the sidebar, and a question is being edited in the right panel.](doc/quizmaster-edit.png?raw=true "Quizmaster Editing a Quiz")

Quizmaster editing a quiz. This allows adding and editing questions, as well as changing the properties of the quiz.

![Screenshot of a quizmaster using QuizDrum to present a quiz live. A panel is visible with the current question. Responses from participants are visible, and quizmaster can award points](doc/quizmaster-present.png?raw=true "Quizmaster Presenting a Quiz")

Quizmaster presenting a quiz live. They can move forward and backward through the list of questions. The page updates with the answers of the participants live, and scores can be awarded. Once scores are submitted, the scoreboard updates.

![Screenshot of a participant using QuizDrum. A panel is visible with the current question. The participant has written an answer for the question](doc/participant-participate.png?raw=true "Participant Answering Questions")

A participant using Quizdrum. As the quizmaster moves through the questions, the page updates with the current question. The participant can submit their answer and update it. If the quizmaster closes responses to a question, the submit button gets disabled.