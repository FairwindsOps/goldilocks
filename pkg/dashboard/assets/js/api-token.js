(function () {
  const apiTokenBoxId = "api-token-box";
  const apiTokenlLabelContentId = "api-token-box__api-token-label-content";
  const apiTokenInputId = "api-token-box__api-token-input";
  const submitBtnId = "api-token-box__submit-btn";

  const apiTokenBox = document.getElementById(apiTokenBoxId);
  const apiTokenLabelContent = document.getElementById(apiTokenlLabelContentId);
  const apiTokenInput = document.getElementById(apiTokenInputId);
  const submitBtn = document.getElementById(submitBtnId);

  const apiKey = localStorage.getItem("apiKey");

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
    if (apiKey) {
      apiTokenBox.style.display = "none";
    }
  }

  apiTokenInput.addEventListener("input", function () {
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
      window.location.reload();
      localStorage.setItem("apiKey", apiTokenInput.value);
    }
  });
})();
