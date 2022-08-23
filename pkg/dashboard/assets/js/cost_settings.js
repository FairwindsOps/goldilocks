(function () {
  const costSettingsBoxId = "cost-settings-box";
  const disableCostSettingsBtnId = "cost-settings__disable-cost-settings";
  const cloudProvidersSelectId = "cost-settings-box__cloud-providers";
  const instanceTypesSelectId = "cost-settings-box__instance-types";
  const submitBtnId = "cost-settings-box__submit-btn";

  const costSettingsBox = document.getElementById(costSettingsBoxId);
  const disableCostSettingsBtn = document.getElementById(
    disableCostSettingsBtnId
  );
  const cloudProvidersSelect = document.getElementById(cloudProvidersSelectId);
  const instanceTypesSelect = document.getElementById(instanceTypesSelectId);
  const submitBtn = document.getElementById(submitBtnId);

  const apiKey = localStorage.getItem("apiKey");
  const isEmailEntered = localStorage.getItem("emailEntered");

  const transformedInstanceTypes = {};
  let selectedCloudProvider = null;

  initUIState();

  function initUIState() {
    if (!apiKey || !isEmailEntered) {
      costSettingsBox.style.display = "none";
    } else {
      loadInstanceTypes();
    }
  }

  function loadInstanceTypes() {
    fetch(
      `${window.INSIGHTS_HOST}/v0/oss/instance-types?ossToken=${apiKey}`
    ).then((response) => {
      if (response) {
        response.json().then(async (data) => {
          await transformInstanceTypes(data);
          await initCloudProvidersUI();
          await initInstanceTypesUI();
        });
      }
    });
  }

  async function transformInstanceTypes(instanceTypes) {
    if (!instanceTypes?.length) return;

    for (const type of instanceTypes) {
      const cloudProvider = type.CloudProvider;

      if (transformedInstanceTypes.hasOwnProperty(cloudProvider)) {
        transformedInstanceTypes[cloudProvider].push(type);
      } else {
        transformedInstanceTypes[cloudProvider] = [type];
      }
    }
  }

  async function initCloudProvidersUI() {
    if (
      !transformedInstanceTypes ||
      !Object.keys(transformedInstanceTypes)?.length
    ) {
      return;
    }
    const cloudProviders = Object.keys(transformedInstanceTypes);
    if (!cloudProviders?.length) {
      return;
    }
    if (cloudProvidersSelect) {
      for (const provider of cloudProviders) {
        cloudProvidersSelect.options[cloudProvidersSelect.options.length] =
          new Option(provider, provider);
      }
    }
    cloudProvidersSelect.options[0].selected = true;
    selectedCloudProvider = cloudProvidersSelect.options[0].value;
  }

  cloudProvidersSelect.addEventListener("change", () => {
    selectedCloudProvider = cloudProvidersSelect.value;
    initInstanceTypesUI();
  });

  async function initInstanceTypesUI() {
    if (
      !transformedInstanceTypes ||
      !Object.keys(transformedInstanceTypes)?.length
    ) {
      return;
    }
    const selectedInstanceTypes =
      transformedInstanceTypes[selectedCloudProvider];
    if (!selectedInstanceTypes?.length) {
      return;
    }
    instanceTypesSelect.options.length = 0;
    const instanceTypeNames = selectedInstanceTypes.map((type) => type.Name);
    for (const name of instanceTypeNames) {
      instanceTypesSelect.options[instanceTypesSelect.options.length] =
        new Option(name, name);
    }
    instanceTypesSelect.options[0].selected = true;
  }

  submitBtn.addEventListener("click", function (e) {
    e.preventDefault();
  });

  disableCostSettingsBtn.addEventListener("click", function () {
    localStorage.removeItem("emailEntered");
    localStorage.removeItem("apiKey");

    window.location.href = window.location.href.split("?")[0];
  });
})();
