function createOption(value, text) {
  const option = document.createElement("option");
  option.value = value;
  option.textContent = text;
  return option;
}

function createPlaceholderOption(text) {
  return createOption("", text);
}

function populateOptions(jsonData) {
  const { srcDirAndItsContents, destination } = jsonData;
    const srcDirs = document.getElementById("sourceDirs");
    const items = document.getElementById("items");
    const destDirs = document.getElementById("destinationDirs");
    const messageElement = document.getElementById("listingMessage");
    messageElement.textContent = "";

    if (jsonData.listingErrors) {
      srcDirs.innerHTML = "";
      items.innerHTML = "";
      destDirs.innerHTML = "";
      messageElement.textContent = "Errors getting directory listing for source or dest";
      messageElement.style.color = "red";
      return;
    }

    srcDirs.innerHTML = "";
    srcDirs.appendChild(createPlaceholderOption("-- select a source directory --"));
    srcDirs.append(...Object.keys(srcDirAndItsContents).map(a => createOption(a, a)));

    items.innerHTML = "";
    srcDirs.onchange = function() {
      items.innerHTML = "";
      items.append(...srcDirAndItsContents[this.value].map(a => createOption(a, a)));
    }

    destDirs.innerHTML = "";
    destDirs.appendChild(createPlaceholderOption("-- select a destination directory --"));
    destDirs.append(...destination.map(a => createOption(a, a)));
}

/*
Check move/copy response
*/
function checkCMResponse(jsonData) {
  populateOptions(jsonData)
  const messageElement = document.getElementById("opMessage");
  if ('opResponse' in jsonData && jsonData.opResponse !== null && jsonData.opResponse.length > 0) {
    messageElement.textContent = JSON.stringify(jsonData);;
    messageElement.style.color = "red";
  } else {
    messageElement.textContent = "Success";
    messageElement.style.color = "green";
  }

}

function refreshOptions() {
  fetch("/data", {
    method: "GET",
    headers: {
      "Accept": "application/json",
    },
  })
  .then(response => response.json())
  .then(jsonData => populateOptions(jsonData))
}

function handleOp(op) {
  // event.preventDefault();
  const src = document.getElementById("sourceDirs").value;
  const items = Array.from(document.getElementById("items").selectedOptions).map(option => option.value);
  const dest = document.getElementById("destinationDirs").value;

  const payload = {
    src : src,
    items: items,
    dest: dest
  };

  fetch("/"+op, {
    method: "POST",
    headers: {
        "Content-Type": "application/json"
    },
    body: JSON.stringify(payload)
  })
  .then(response => response.json())
  .then(jsonData => checkCMResponse(jsonData))

}


window.onload = function() {
  refreshOptions();
  const moveButton = document.getElementById("moveButton");
  const copyButton = document.getElementById("copyButton");
  const form = document.getElementById("moveForm");
  form.addEventListener("submit", function(event) {
    event.preventDefault();
    if (event.submitter === moveButton) {
      handleMove("move");
    } else if (event.submitter === copyButton) {
      handleCopy("copy");
    }
  });
}
