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

function submitansClick(e) {
  const formElement = document.getElementById('ansform');
  const infoElement = document.getElementById('info');
  const data = new URLSearchParams(new FormData(formElement));

  posty('/api/participant/submit-answer', data)
    .then(response => { return response.json(); })
    .then(myBlob => {
      document.getElementById('ans-id').value = myBlob;
      info.innerHTML = "Saved.";
    })
    .catch(showError);
}

async function updateOnStatus() {
  const qzid = parseInt(document.getElementById('qz-id').value);
  const qnid = parseInt(document.getElementById('qn-id').value)
  const sbtn = document.getElementById('submitans');
  const info = document.getElementById('info');
  let success = false;
  await getty('/api/participant/quiz/' + qzid + '/getstatus')
    .then(r => { return r.json(); })
    .then(j => {
      success = true;
      if (j && j.QuestionID && (j.QuestionID != qnid)) {
        location.reload();
        return;
      }
      if (j && j.hasOwnProperty('AcceptingResponses')) {
        sbtn.disabled = !j.AcceptingResponses;
      }
    })
    .catch(showError);

  if (success) {
    currentTimeout = 700;
  } else {
    currentTimeout = Math.floor(1.3 * currentTimeout);
    if (currentTimeout > 3000) {
      info.innerHTML = "Something went wrong, we can't contact the server. Reloading the page."
      location.reload();
      return;
    }
  }
  window.setTimeout(updateOnStatus, currentTimeout);
}

function btncrtClick(e) {
  const formElement = document.getElementById('form-set-profile');
  const infoElement = document.getElementById('info');
  const data = new URLSearchParams(new FormData(formElement));

  posty('/api/participant/set-profile', data)
    .then(response => { return response.blob(); })
    .then(myBlob => {
      info.innerHTML = "Saved.";
      window.location.href = "/participant/quiz/{{.Q.GetId}}/live";
    })
    .catch(showError);
}