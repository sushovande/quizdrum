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

// r can be either an Error or a Promise<Response>
async function showError(r) {
  const info = document.getElementById('info')
  if (r.text) {
    const sts = r.status;
    r.text().then(msg => {
      info.innerHTML = 'Error HTTP (' + sts + "): " + msg;
    });
  } else {
    info.innerHTML = 'Error: ' + r.message;
  }
}

function setupMaterial() {
  for (let a of document.querySelectorAll('.mdc-button')) {
    mdc.ripple.MDCRipple.attachTo(a);
  }
  for (let a of document.querySelectorAll('.mdc-text-field')) {
    const textField = new mdc.textField.MDCTextField(a);
  }
  for (let ff of document.querySelectorAll('.mdc-form-field')) {
    const rad = ff.querySelector('.mdc-radio');
    if (rad) {
      const fld = new mdc.formField.MDCFormField(ff);
      const rdo = new mdc.radio.MDCRadio(rad);
      fld.input = rdo;
    }
  }
  for (let tt of document.querySelectorAll('.mdc-data-table')) {
    new mdc.dataTable.MDCDataTable(tt);
  }
}

function qmAnsTimestampReplace() {
  const options = {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: 'numeric',
    second: 'numeric',
    timeZoneName: 'short'
  };
  let fmt;
  if ((new Date()).getTimezoneOffset() < 0) {
    fmt = new Intl.DateTimeFormat('en-IN', options);
  } else {
    fmt = new Intl.DateTimeFormat('en-US', options);
  }
  const allAns = document.querySelectorAll('.anstime');
  for (let a of allAns) {
    if (a.dataset && a.dataset['timestamp']) {
      let d = new Date(parseInt(a.dataset['timestamp']) * 1000);
      a.innerHTML = fmt.format(d);
    }
  }
}

async function posty(url, data) {
  return fetch(url, {
    method: 'post',
    body: data,
  })
  .then(response => {
    if (!response.ok) {
      throw response;
    }
    return response;
  })
}

// Sends a POST request and returns the json. Performs error handling for you.
async function postj(url, data) {
  return fetch(url, {
    method: 'post',
    body: data,
  })
  .then(response => {
    if (!response.ok) {
      throw response;
    }
    return response;
  })
  .then( r => r.json())
  .catch(e => { showError(e); throw e; });
}

// Sends a GET request to URL and returns the response.
// Caller is responsible for calling .text(), .json() or .blob() on the response.
// Caller is responsible for calling .catch(showError) on the response.
async function getty(url) {
  return fetch(url)
  .then(response => {
    if (!response.ok) {
      throw response;
    }
    return response;
  })
}

// Sends a GET request and returns the json. Performs error handling for you.
async function getj(url) {
  return fetch(url)
    .then(response => {
      if (!response.ok) {
        throw response;
      }
      return response;
    })
    .then(r => r.json())
    .catch(e => { showError(e); throw e; });
}

async function putj(url, data) {
  return fetch(url, {
    method: 'put',
    body: data,
  })
    .then(response => {
      if (!response.ok) {
        throw response;
      }
      return response;
    })
    .then(r => r.json())
    .catch(e => { showError(e); throw e; });
}

async function putt(url, data) {
  return fetch(url, {
    method: 'put',
    body: data,
  })
    .then(response => {
      if (!response.ok) {
        throw response;
      }
      return response;
    })
    .then(r => r.text())
    .catch(e => { showError(e); throw e; });
}

async function deletet(url) {
  return fetch(url, {
    method: 'delete',
  })
    .then(response => {
      if (!response.ok) {
        throw response;
      }
      return response;
    })
    .then(r => r.text())
    .catch(e => { showError(e); throw e; });
}