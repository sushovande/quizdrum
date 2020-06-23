# QuizDrum
QuizDrum is a piece of software that allows you to host a website for conducting and participating in quizzes. Quizmasters can log on to your QuizDrum website with their Google account and create new quizzes, add and modify questions, and host a live quizzing session. Participants can log on to your QuizDrum website and answer and update questions and keep track of the scoreboard. The quizmaster can awards points to each answer, and control the flow of the quiz.

You can install QuizDrum on most linux and windows machines. It is written in Go and JavaScript, and stores its data using SQLite. There are no pre-set limits on the number of participants.

*This is not an officially supported Google product.*

## Installation
Because of the weird way SQLite and go works, SQLite is not listed as a dependency in the `go.mod` file. Please see the [instructions](https://github.com/mattn/go-sqlite3#installation) on installing SQLite first, before proceeding further. Particularly on Windows, where having `gcc` in the path is uncommon, you may have to install the 64-bit `gcc` from a source like [tdm-gcc](https://jmeubank.github.io/tdm-gcc/).
1. Confirm you have installed SQLite as described above
2. `go get github.com/sushovande/quizdrum`

In order to allow quizmasters to log in, you will need to set up a client ID so that they can log in with their Google Account. To do so, follow the steps in the [Create Authorization Credentials](https://developers.google.com/identity/sign-in/web/sign-in#create_authorization_credentials) section, and save the client id. You can then run quizdrum as
```
quizdrum --port=80 --oauth_client_id=<your-google-client-id>
```
You could also [set it up as a service](https://medium.com/@benmorel/creating-a-linux-service-with-systemd-611b5c8b91d6). You can also run it on a different port and then set up [a reverse proxy to it](https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/).

## Use
Quizmasters should log in with their Google Account. Once logged in, they can create a new quiz from the homepage. On the quiz editing page, they can add, remove, and modify questions, as well as change the name and description of the quiz. Questions can of five types: short text answers, multiple choice, true or false, numeric answers with integers, and decimal numbers.

Once the quizmaster is satisfied with the quiz, they can go to the homepage and press the **Present** button to begin the quiz session. Participants can visit the quiz website and log in (either with their Google Account, or as a Guest). They need to click the **Participate** button next to the quiz to participate in, and then they can choose a screen name for that particular quiz.

As the quizmaster proceeds through the quiz, they can move back and forth over questions. They can stop accepting answers at any time, and awards points to the answers they receive. The page will update automatically with answers as they are submitted and updated. The quizmaster will have to click **Save Scores** to finalize awarding the points, at which point the scoreboard will update with the scores.

