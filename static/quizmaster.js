/**
 * Copyright 2020 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

 function btncrtClick(e) {
  const formElement = document.getElementById('qn-new-form');
  const infoElement = document.getElementById('info');
  const data = new URLSearchParams(new FormData(formElement));

  const existingQnId = document.getElementById('qn-id').value
  if (existingQnId == '') {
    posty('/api/quizmaster/question/new', data)
      .then(r => { return r.json(); })
      .then(j => {
        info.innerHTML = "Saved.";
        addQuestionWithId(j);
        resetForm();
      })
      .catch(showError);
  } else {
    putt('/api/quizmaster/question/' + existingQnId + '/update', data)
      .then(j => {
        info.innerHTML = "Saved.";
        const ctr = findQnContainerWithId(existingQnId);
        if (ctr) {
          ctr.querySelector('.question-title').innerHTML = data.get("qn-title");
          ctr.querySelector('.question-title').title = data.get("qn-title");
          ctr.querySelector('.question-subtitle').innerHTML = data.get("qn-body");
          ctr.querySelector('.question-subtitle').title = data.get("qn-body");
        }
      });
  }
}

function addQuestionWithId(id) {
  const tmplElem = document.getElementById('qn-container-template');
  const newQnElem = tmplElem.content.cloneNode(true);
  const qnContainerElem = newQnElem.querySelector('.question-container');
  qnContainerElem.dataset['qnid'] = id;

  const titleElem = newQnElem.querySelector('.question-title');
  const titleValue = document.getElementById('qn-title').value;
  titleElem.innerHTML = titleValue;
  titleElem.title = titleValue;

  const descrElem = newQnElem.querySelector('.question-subtitle');
  const bodyValue = document.getElementById('qn-body').value;
  descrElem.innerHTML = bodyValue;
  descrElem.title = bodyValue;

  const parentElem = document.getElementById('qn-pane-parent');
  parentElem.insertBefore(newQnElem, tmplElem);
}

function questionClicked(e) {
  switchToQuestionPane();
  if (!e.dataset.qnid) {
    document.getElementById('info').innerHTML = "Could not figure out the question ID you wanted.";
    return;
  }
  getj('/api/quizmaster/question/' + e.dataset.qnid)
    .then(j => {
      populateQuestionForm(j);
    })
}

function addqnClick(e) {
  switchToQuestionPane();
  resetForm();
  document.getElementById('btncrt-label').innerHTML = 'Add Question';
}

function delbtnClick(e) {
  const qnid = document.getElementById('qn-id').value;
  let container = findQnContainerWithId(qnid);

  deletet('/api/quizmaster/question/' + qnid + '/delete')
    .then(t => {
      if (container) {
        container.remove();
      }
      resetForm();
    })
}

function populateQuestionForm(j) {
  /*
  Note: j is of the form
  {"id":"13","quizId":"7","title":"Question 1","htmlBody":
  "What is the correct number of sds.","type":"MULTIPLE_CHOICE_ANSWER",
  "choices":[{"htmlBody":"sd1"},{"htmlBody":"sd2"},{"htmlBody":"sd3"}]}
  */
  let e = new Event('input', { bubbles: true, cancelable: true });
  document.getElementById('qn-id').value = j.id;
  document.getElementById('quiz-id').value = j.quizId;
  document.getElementById('qn-title').value = j.title;
  document.getElementById('qn-title').dispatchEvent(e);
  document.getElementById('qn-body').value = j.htmlBody;
  document.getElementById('qn-body').dispatchEvent(e);

  let tp = document.getElementById('qn-new-type-text');
  switch (j.type) {
    case "TEXT_ANSWER":
      tp = document.getElementById('qn-new-type-text');
      break;
    case "LONG_TEXT_ANSWER":
      tp = document.getElementById('qn-new-type-longtext');
      break;
    case "INT64_ANSWER":
      tp = document.getElementById('qn-new-type-int');
      break;
    case "FLOAT_ANSWER":
      tp = document.getElementById('qn-new-type-float');
      break;
    case "BOOL_ANSWER":
      tp = document.getElementById('qn-new-type-bool');
      break;
    case "MULTIPLE_CHOICE_ANSWER":
      tp = document.getElementById('qn-new-type-mcq');
      break;
    default:
      document.getElementById('info').innerHTML = "Got a weird question type";
      break;
  }
  tp.checked = true;
  tp.dispatchEvent(e);

  removeAllMcqRows();
  if (j.choices && j.type == "MULTIPLE_CHOICE_ANSWER") {
    for (let i = 0; i < j.choices.length; i++) {
      if (i == 0) {
        document.getElementById('mcq-inp-1').value = j.choices[0].htmlBody;
      } else {
        addMcqRowWithText(j.choices[i].htmlBody);
      }
    }
  }
  document.getElementById('btncrt-label').innerHTML = "Update Question";
  document.getElementById('btndel').disabled = false;
}

function resetForm() {
  document.getElementById('qn-id').value = '';
  document.getElementById('qn-title').value = '';
  document.getElementById('qn-body').value = '';
  document.getElementById('qn-new-type-text').checked = true;
  removeAllMcqRows();
  qnTypeChanged(document.getElementById('qn-new-type-text'));
  let e = new Event('blur', { bubbles: true, cancelable: true });
  document.getElementById('qn-title').dispatchEvent(e);
  document.getElementById('qn-body').dispatchEvent(e);
  document.getElementById('btndel').disabled = true;
}

function textinput(e) {
  const indexOfTypedInput = parseInt(e.id.substring(8)) - 1;
  const ma = document.getElementById("mcq-author");
  if (indexOfTypedInput < ma.children.length - 1) {
    // this is not the last element
    return;
  }
  addMcqRowWithText('');
}

function addMcqRowWithText(txt) {
  const ma = document.getElementById("mcq-author");
  const template = document.getElementById('mcq-template');
  let clone = template.content.cloneNode(true);
  const idx = ma.children.length + 1;
  clone.firstElementChild.id = 'mcq-option-' + idx;
  clone.querySelector('strong').innerHTML = 'Option ' + idx + ':';
  clone.querySelector('input').id = 'mcq-inp-' + idx;
  clone.querySelector('input').value = txt;
  clone.querySelector('button').id = 'mcq-btn-' + idx;
  ma.appendChild(clone);
}

function removeAllMcqRows() {
  document.getElementById('mcq-inp-1').value = "";
  const ma = document.getElementById("mcq-author");
  // We have to loop from the bottom up, otherwise the textinput event will fire
  // and add more items (i think).
  for (let i = ma.children.length; i > 1; i--) {
    document.getElementById('mcq-option-' + i).remove();
  }
}

function removerow(e) {
  // We're calling this 'ordinal' because it is 1-based
  const ordinalClicked = parseInt(e.id.substring(8));
  const ma = document.getElementById("mcq-author");
  if (ma.children.length == 1) {
    // clicked the only item, just clear the input
    document.getElementById('mcq-inp-1').value = "";
    return;
  }
  // Collect the list of strings
  szMcqTexts = [];
  for (let i = ordinalClicked + 1; i <= ma.children.length; i++) {
    szMcqTexts.push(document.getElementById('mcq-inp-' + i).value);
  }
  for (let i = 0; i < szMcqTexts.length; i++) {
    document.getElementById('mcq-inp-' + (ordinalClicked + i)).value = szMcqTexts[i];
  }
  document.getElementById('mcq-option-' + ma.children.length).remove();
}

function qnTypeChanged(e) {
  const mcqOption = document.getElementById('qn-new-type-mcq');
  const authorDiv = document.getElementById('mcq-author-container');
  if (mcqOption.checked) {
    authorDiv.style.display = 'block';
  } else {
    authorDiv.style.display = 'none';
  }
}

function findQnContainerWithId(qnid) {
  for (let e of document.querySelectorAll('.question-container')) {
    if (e.dataset.qnid && e.dataset.qnid == qnid) {
      return e;
    }
  }
}

function switchToQuestionPane() {
  document.getElementById('qzformpane').style.display = 'none';
  document.getElementById('qnformpane').style.display = 'block';
}

function switchToQuizPane() {
  document.getElementById('qnformpane').style.display = 'none';
  document.getElementById('qzformpane').style.display = 'block';
}

function quizClicked() {
  switchToQuizPane();
}

function btnqzupdateClick(e) {
  const data = new URLSearchParams(new FormData(document.getElementById('qz-form')));
  const qzid = parseInt(document.getElementById('qz-id').value);
  putt('/api/quizmaster/quiz/' + qzid + '/updateproperties', data)
    .then(j => { document.getElementById('info').innerHTML = "Updated Quiz."; })
}

function btnqzdelClick(e) {
  const qzid = parseInt(document.getElementById('qz-id').value);
  deletet('/api/quizmaster/quiz/' + qzid + '/delete')
    .then(t => {
      document.getElementById('info').innerHTML = "Deleted Quiz.";
      document.getElementById('btnqzdel-container').style.display = 'none'
      document.getElementById('btnqzundodel-container').style.display = 'inline'
    })
}

function btnqzundodelClick(e) {
  const data = new URLSearchParams({});
  const qzid = parseInt(document.getElementById('qz-id').value);
  putj('/api/quizmaster/quiz/' + qzid + '/reinstate', data)
    .then(t => {
      document.getElementById('info').innerHTML = "Reinstated Quiz.";
      document.getElementById('btnqzundodel-container').style.display = 'none'
      document.getElementById('btnqzdel-container').style.display = 'inline'
    })
}


function btn_nextqClick(e) {
  const curId = parseInt(document.getElementById('qn-id').value);
  const qnIndex = allQuestionIds.indexOf(curId);
  const nextIndex = qnIndex + 1;
  sendCurrentQuestion(nextIndex);
}

function btn_prevqClick(e) {
  const curId = parseInt(document.getElementById('qn-id').value);
  const qnIndex = allQuestionIds.indexOf(curId);
  const nextIndex = qnIndex - 1;
  sendCurrentQuestion(nextIndex);
}

function sendCurrentQuestion(nextIndex) {
  const qzId = parseInt(document.getElementById('qz-id').value);
  const info = document.getElementById('info');
  if (nextIndex >= allQuestionIds.length || nextIndex < 0) {
    // handle scorecard instead
  } else {
    const data = new URLSearchParams({});
    posty('/api/quizmaster/quiz/' + qzId + '/setactive/' + allQuestionIds[nextIndex], data)
      .then(r => { return r.blob(); })
      .then(myBlob => {
        info.innerHTML = "Next ID has been marked as active. Refresh the page now.";
        location.reload();
      })
      .catch(showError);
  }
}

async function btn_refreshansClick(e) {
  const curId = parseInt(document.getElementById('qn-id').value);
  const info = document.getElementById('info');
  let success = false;
  await getty('/api/quizmaster/question/' + curId + '/getallanswers')
    .then(r => { return r.text(); })
    .then(myText => {
      document.getElementById('answercontainer').innerHTML = myText;
      qmAnsTimestampReplace();
      repopulateSavedScores();
      success = true;
    })
    .catch(showError);

  if (success) {
    currentTimeout = 3100;
  } else {
    currentTimeout = Math.floor(1.3 * currentTimeout);
    if (currentTimeout > 10000) {
      currentTimeout = 10000;
      document.getElementById('info').textContent = 'Something\'s wrong. Can\'t seem to fetch answers from the server';
    }
  }
  if (keepRefreshingScores) {
    window.setTimeout(btn_refreshansClick, currentTimeout, {});
  }
}

function btn_btnscoreClick(e) {
  const curId = parseInt(document.getElementById('qn-id').value);
  const info = document.getElementById('info');
  const formElement = document.getElementById('scoringform');
  const data = new URLSearchParams(new FormData(formElement));
  posty('/api/quizmaster/question/' + curId + '/savescores', data)
    .then(r => { return r.text(); })
    .then(t => {
      info.innerHTML = "Scores Saved.";
    })
    .catch(showError);
}

function btn_stopansClick(e) {
  const qzId = parseInt(document.getElementById('qz-id').value);
  const btnStopAns = document.getElementById('stopans');
  const stopAnsLabel = document.getElementById('stopanslabel');
  const info = document.getElementById('info');
  const data = new URLSearchParams();
  let ar = false;

  if (stopAnsLabel.innerHTML == "Stop Accepting Responses") {
    data.append("ar", "false");
  } else {
    data.append("ar", "true");
    ar = true;
  }

  posty('/api/quizmaster/quiz/' + qzId + '/setacceptingresponses', data)
    .then(r => { return r.text(); })
    .then(t => {
      if (ar) {
        stopAnsLabel.innerHTML = "Stop Accepting Responses";
        keepRefreshingScores = true;
      } else {
        stopAnsLabel.innerHTML = "Accept Responses Again";
        keepRefreshingScores = false;
      }
      btn_refreshansClick({});
    })
    .catch(showError);
}

var savedScores;
function scoreChanged(e) {
  const formElement = document.getElementById('scoringform');
  savedScores = new FormData(formElement);
  if (e.id.endsWith('-score-custom')) {
    const tElem = document.getElementById(e.id.replace('-score-custom', '-custom-score'));
    tElem.focus();
  }
}

function repopulateSavedScores() {
  if (!savedScores) {
    return;
  }
  for (let kvpair of savedScores) {
    const k = kvpair[0];
    const v = kvpair[1];
    // We get two types of keys here: ans-<ans_id>-score, and ans-<ans_id>-custom-score.
    if (k.endsWith('-custom-score')) {
      const textElem = document.getElementById(k);
      if (textElem) {
        textElem.value = v;
      }
      continue;
    }
    let formElem = document.getElementById(k + '-' + v);
    if (formElem) {
      formElem.checked = true;
    }
  }
}