(function () {
  const apiTokenBoxId = "api-token-box";
  const disableCostSettingsBtnId = "api-token__disable-cost-settings";
  const apiTokenlLabelContentId = "api-token-box__api-token-label-content";
  const apiTokenInputId = "api-token-box__api-token-input";
  const apiTokenInputErrorId = "api-token-box__input-error";
  const submitBtnId = "api-token-box__submit-btn";

  const apiTokenBox = document.getElementById(apiTokenBoxId);
  const disableCostSettingsBtn = document.getElementById(
    disableCostSettingsBtnId
  );
  const apiTokenLabelContent = document.getElementById(apiTokenlLabelContentId);
  const apiTokenInput = document.getElementById(apiTokenInputId);
  const apiTokenInputError = document.getElementById(apiTokenInputErrorId);
  const submitBtn = document.getElementById(submitBtnId);

  const apiKey = localStorage.getItem("apiKey");
  const isEmailEntered = localStorage.getItem("emailEntered");

  const urlParams = new URLSearchParams(window.location.search);

  initQueryParams();
  initUIState();

  function initQueryParams() {
    if (apiKey && !urlParams.has("apiKey")) {
      setQueryParam("apiKey", apiKey);
    }
  }

  function setQueryParam(key, value) {
    urlParams.set(key, value);
    window.location.search = urlParams;
  }

  function initUIState() {
    if (apiKey || !isEmailEntered) {
      apiTokenBox.style.display = "none";
    }
  }

  apiTokenInput.addEventListener("input", function () {
    apiTokenInputError.style.display = "none";
    toggleLabelContent(this.value);
  });

  function toggleLabelContent(inputApiToken) {
    apiTokenLabelContent.style.display = inputApiToken
      ? "none"
      : "inline-block";
  }

  submitBtn.addEventListener("click", function (e) {
    e.preventDefault();

    if (apiTokenInput.validity.valid) {
      const inputApiToken = apiTokenInput.value.trim();
      fetch(
        `${window.INSIGHTS_HOST}/v0/oss/instance-types?ossToken=${inputApiToken}`
      ).then((response) => {
        if (response && response.status !== 400) {
          window.location.reload();
          localStorage.setItem("apiKey", apiTokenInput.value.trim());
        } else {
          apiTokenInputError.style.display = "block";
        }
      });
    }
  });

  disableCostSettingsBtn.addEventListener("click", function () {
    localStorage.removeItem("emailEntered");
    localStorage.removeItem("apiKey");

    window.location.href = window.location.href.split("?")[0];
  });
})();
