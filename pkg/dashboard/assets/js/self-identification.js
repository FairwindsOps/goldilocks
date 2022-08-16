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
    const checked = emailCheckbox.checked;
    toggleLabelContent(this.value);
    toggleSubmitBtn(checked);
  });

  emailCheckbox.addEventListener("change", function () {
    toggleSubmitBtn(this.checked);
  });

  function toggleLabelContent(inputEmail) {
    emailLabelContent.style.display = inputEmail ? "none" : "inline-block";
  }

  function toggleSubmitBtn(checked) {
    if (isInputInfoValid(checked)) {
      submitBtn.disabled = false;
      submitBtn.classList.add("email-box__submit-btn--active");
    } else {
      submitBtn.disabled = true;
      submitBtn.classList.remove("email-box__submit-btn--active");
    }
  }

  submitBtn.addEventListener("click", function (e) {
    e.preventDefault();
    const email = emailInput.value;
    const checked = emailCheckbox.checked;
    if (isInputInfoValid(email, checked)) {
      fetch(`${window.INSIGHTS_HOST}/v0/oss/users`, {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ email, project: "goldilocks" }),
      }).then((response) => {
        if (response) {
          response.json().then((data) => {
            if (data?.token) {
              window.location.reload();
              localStorage.setItem("emailEntered", true);
            }
          });
        }
      });
    }
  });

  function isInputInfoValid(checked) {
    return checked && emailInput.validity.valid;
  }
})();
