(function () {
  const emailBoxId = "email-box";
  const emailLabelContentId = "email-box__email-label-content";
  const emailInputId = "email-box__email-input";
  const emailCheckboxId = "email-box__checkbox";
  const submitBtnId = "email-box__submit-btn";

  const emailBox = document.getElementById(emailBoxId);
  const emailLabelContent = document.getElementById(emailLabelContentId);
  const emailInput = document.getElementById(emailInputId);
  const emailCheckbox = document.getElementById(emailCheckboxId);
  const submitBtn = document.getElementById(submitBtnId);

  const urlParams = new URLSearchParams(window.location.search);

  initQueryParams();
  initUIState();

  function initQueryParams() {
    const isEmailEntered = localStorage.getItem("emailEntered");
    if (isEmailEntered && !urlParams.has("emailEntered")) {
      setQueryParam("emailEntered", "true");
    }
  }

  function setQueryParam(key, value) {
    urlParams.set(key, value);
    window.location.search = urlParams;
  }

  function initUIState() {
    if (urlParams.get("emailEntered")) {
      emailBox.style.display = "none";
    }
  }

  emailInput.addEventListener("input", function (evt) {
    toggleLabelContent(this.value);
  });

  function toggleLabelContent(inputEmail) {
    emailLabelContent.style.display = inputEmail ? "none" : "inline-block";
  }

  submitBtn.addEventListener("click", function () {
    if (emailCheckbox.checked && emailInput.validity.valid) {
      // TODO: write logic to call the API to get the access token.
      window.location.reload();
      localStorage.setItem("emailEntered", true);
    }
  });
})();
