(function () {
  const emailBoxId = "email-box";
  const emailInputId = "email-box__email-input";
  const emailCheckboxId = "emailbox__checkbox";
  const submitBtnId = "email-box__submit-btn";

  const emailBox = document.getElementById(emailBoxId);
  const emailInput = document.getElementById(emailInputId);
  const emailCheckbox = document.getElementById(emailCheckboxId);
  const submitBtn = document.getElementById(submitBtnId);

  const urlParams = new URLSearchParams(window.location.search);

  initQueryParams();
  initUIState();

  function initQueryParams() {
    const enteredEmail = localStorage.getItem("enteredEmail");
    if (enteredEmail && !urlParams.has("emailEntered")) {
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
    const checked = emailCheckbox.checked;
    toggleSubmitBtn(this.value, checked);
  });

  emailCheckbox.addEventListener("change", function () {
    const email = emailInput.value;
    toggleSubmitBtn(email, this.checked);
  });

  function toggleSubmitBtn(email, checked) {
    if (isInputInfoValid(email, checked)) {
      submitBtn.disabled = false;
      submitBtn.classList.add("email-box__submit-btn--active");
    } else {
      submitBtn.disabled = true;
      submitBtn.classList.remove("email-box__submit-btn--active");
    }
  }

  submitBtn.addEventListener("click", function () {
    const email = emailInput.value;
    const checked = emailCheckbox.checked;
    if (isInputInfoValid(email, checked)) {
      // TODO: write logic to call the API to get the access token.
      window.location.reload();
      localStorage.setItem("enteredEmail", email);
    }
  });

  function isInputInfoValid(email, checked) {
    return checked && validator.isEmail(email);
  }
})();
