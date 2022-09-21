(function () {
  const costSettingsBoxId = "cost-settings-box";
  const disableCostSettingsBtnId = "cost-settings__disable-cost-settings";
  const cloudProvidersSelectId = "cost-settings-box__cloud-providers";
  const instanceTypesSelectId = "cost-settings-box__instance-types";
  const costPerCPUInputId = "cost-settings-box__cost-per-cpu-input";
  const costPerGBInputId = "cost-settings-box__cost-per-gb-input";
  const submitBtnId = "cost-settings-box__submit-btn";
  const lowerNumberClass = "lower-number";

  const costSettingsBox = document.getElementById(costSettingsBoxId);
  const disableCostSettingsBtn = document.getElementById(
    disableCostSettingsBtnId
  );
  const cloudProvidersSelect = document.getElementById(cloudProvidersSelectId);
  const instanceTypesSelect = document.getElementById(instanceTypesSelectId);
  const costPerCPUInput = document.getElementById(costPerCPUInputId);
  const costPerGBInput = document.getElementById(costPerGBInputId);
  const submitBtn = document.getElementById(submitBtnId);
  const lowerNumbers = document.getElementsByClassName(lowerNumberClass);

  const apiKeyLS = "apiKey";
  const emailEnteredKey = "emailEntered";
  const selectedCloudProviderKey = "selectedCloudProvider";
  const selectedInstanceTypeKey = "selectedInstanceType";
  const costPerCPUKey = "costPerCPU";
  const costPerGBKey = "costPerGB";
  const otherOption = "Other";
  const emptyString = "";
  const defaultCloudProvider = "AWS";
  let defaultInstanceType = "691";
  const transformedInstanceTypes = {};

  const apiKey = localStorage.getItem(apiKeyLS);
  const isEmailEntered = localStorage.getItem(emailEnteredKey);
  const currentCostPerCPU = localStorage.getItem(costPerCPUKey);
  const currentCostPerGB = localStorage.getItem(costPerGBKey);

  const urlParams = new URLSearchParams(window.location.search);

  let selectedCloudProvider = localStorage.getItem(selectedCloudProviderKey);

  let selectedInstanceType = localStorage.getItem(selectedInstanceTypeKey);

  initQueryParams();
  initUIState();
  initLowerNumberUI();

  function initQueryParams() {
    if (currentCostPerCPU && !urlParams.has(costPerCPUKey)) {
      setQueryParam(costPerCPUKey, currentCostPerCPU);
    }
    if (currentCostPerGB && !urlParams.has(costPerGBKey)) {
      setQueryParam(costPerGBKey, currentCostPerGB);
    }
  }

  function setQueryParam(key, value) {
    urlParams.set(key, value);
    window.location.search = urlParams;
  }

  function initUIState() {
    if (!apiKey || !isEmailEntered) {
      costSettingsBox.style.display = "none";
      return;
    }
    loadInstanceTypes();
  }

  function initLowerNumberUI() {
    if (!lowerNumbers) {
      return;
    }
    Array.from(lowerNumbers).forEach((lowerNumber) => {
      lowerNumber.style.display =
        lowerNumber && currentCostPerCPU && currentCostPerGB
          ? "inline"
          : "none";
    });
  }

  function loadInstanceTypes() {
    fetch(
      `${window.INSIGHTS_HOST}/v0/oss/instance-types?ossToken=${apiKey}`
    ).then((response) => {
      if (response) {
        response.json().then(async (data) => {
          await transformInstanceTypes(data);
          await getDefaultInstanceType();
          await initCloudProvidersUI();
          await initSelectedCloudProvider();
          await initInstanceTypesUI();
          await initSelectedInstanceType();
          await updateInputs();
        });
      }
    });
  }

  async function getDefaultInstanceType() {
    t3Medium = transformedInstanceTypes[defaultCloudProvider].find(
      (instanceType) => instanceType.Name === "t3.medium"
    );
    if (t3Medium?.ID) {
      defaultInstanceType = String(t3Medium?.ID);
    }
  }

  async function transformInstanceTypes(instanceTypes) {
    if (!instanceTypes?.length) {
      return;
    }
    for (const type of instanceTypes) {
      const cloudProvider = type.CloudProvider;
      transformedInstanceTypes.hasOwnProperty(cloudProvider)
        ? transformedInstanceTypes[cloudProvider].push(type)
        : (transformedInstanceTypes[cloudProvider] = [type]);
    }
  }

  async function initCloudProvidersUI() {
    if (!shouldInit()) {
      return;
    }
    const cloudProviders = Object.keys(transformedInstanceTypes);
    for (const provider of cloudProviders) {
      cloudProvidersSelect.options[cloudProvidersSelect.options.length] =
        new Option(provider, provider);
    }
  }

  cloudProvidersSelect.addEventListener("change", async () => {
    selectedCloudProvider = cloudProvidersSelect.value;
    await initSelectedCloudProvider();
    await initInstanceTypesUI();
    await initSelectedInstanceType();
    await updateInputs();
  });

  async function initSelectedCloudProvider() {
    if (!selectedCloudProvider) {
      initDefaultCloudProvider();
      return;
    }
    if (selectedCloudProvider !== otherOption) {
      setSelectedOptionUI(cloudProvidersSelect, selectedCloudProvider);
      return;
    }
    setSelectedOptionUI(cloudProvidersSelect, otherOption);
    selectedCloudProvider = otherOption;
  }

  function initDefaultCloudProvider() {
    for (const option of cloudProvidersSelect) {
      if (option.value === defaultCloudProvider) {
        selectedCloudProvider = defaultCloudProvider;
        option.selected = true;
      }
    }
  }

  instanceTypesSelect.addEventListener("change", async () => {
    selectedInstanceType = instanceTypesSelect.value;
    await initSelectedInstanceType();
    await updateInputs();
  });

  async function initInstanceTypesUI() {
    if (!shouldInit()) {
      return;
    }
    instanceTypesSelect.options.length = 0;
    const sortedInstanceTypes = sortInstanceTypes(
      transformedInstanceTypes[selectedCloudProvider]
    );
    for (const type of sortedInstanceTypes) {
      instanceTypesSelect.options[instanceTypesSelect.options.length] =
        new Option(type.Name, type.ID);
    }
  }

  function sortInstanceTypes(instanceTypes) {
    if (!instanceTypes?.length) {
      return [];
    }
    return instanceTypes.sort((a, b) => a.Name.localeCompare(b.Name));
  }

  function shouldInit() {
    return (
      transformedInstanceTypes && Object.keys(transformedInstanceTypes)?.length
    );
  }

  async function initSelectedInstanceType() {
    if (!selectedInstanceType) {
      initDefaultInstanceType();
      return;
    }
    if (selectedInstanceType !== otherOption) {
      setSelectedOptionUI(instanceTypesSelect.options, selectedInstanceType);
      return;
    }
    instanceTypesSelect.options[0].selected = true;
  }

  function initDefaultInstanceType() {
    for (const option of instanceTypesSelect) {
      if (option.value === defaultInstanceType) {
        selectedInstanceType = option.value;
        option.selected = true;
      }
    }
  }

  function setSelectedOptionUI(options, selectedOption) {
    for (const option of options) {
      if (option.value === selectedOption) {
        option.selected = true;
        return;
      }
    }
  }

  async function updateInputs() {
    displayInputValues(emptyString, emptyString);
    if (selectedCloudProvider === otherOption) {
      updateInputsOtherOption();
    } else {
      updateInputsCloudOption();
    }
  }

  function updateInputsOtherOption() {
    toggleInputs(false);
    const costPerCPU = localStorage.getItem(costPerCPUKey);
    const costPerGB = localStorage.getItem(costPerGBKey);
    displayInputValues(costPerCPU, costPerGB);
  }

  function displayInputValues(costPerCPU, costPerGB) {
    costPerCPUInput.value = costPerCPU;
    costPerGBInput.value = costPerGB;
  }

  function updateInputsCloudOption() {
    toggleInputs(true);
    const selectedInstanceType = getInstanceTypeByID();
    if (selectedInstanceType) {
      const costPerCPU = calculateCostPerCPU(selectedInstanceType);
      const costPerGB = calculateCostPerGB(selectedInstanceType);
      displayInputValues(costPerCPU, costPerGB);
    }
  }

  function toggleInputs(isDisabled) {
    costPerCPUInput.disabled = isDisabled;
    costPerGBInput.disabled = isDisabled;
  }

  function getInstanceTypeByID() {
    return transformedInstanceTypes[selectedCloudProvider].find(
      (type) => type.ID === +instanceTypesSelect.value
    );
  }

  function calculateCostPerGB(selectedInstanceType) {
    if (!selectedInstanceType) {
      return;
    }
    return (
      selectedInstanceType.CostPerNode /
      2 /
      selectedInstanceType.AmountOfMemory
    ).toFixed(4);
  }

  function calculateCostPerCPU(selectedInstanceType) {
    if (!selectedInstanceType) {
      return;
    }
    return (
      selectedInstanceType.CostPerNode /
      2 /
      selectedInstanceType.NumOfCpus
    ).toFixed(4);
  }

  submitBtn.addEventListener("click", function (e) {
    e.preventDefault();
    if (selectedCloudProvider === otherOption) {
      saveOption(otherOption, otherOption);
    } else {
      saveOption(selectedCloudProvider, selectedInstanceType);
    }
  });

  function saveOption(cloudProvider, instanceType) {
    const costPerCPU = costPerCPUInput.value;
    const costPerGB = costPerGBInput.value;
    localStorage.setItem(selectedCloudProviderKey, cloudProvider);
    localStorage.setItem(selectedInstanceTypeKey, instanceType);
    localStorage.setItem(costPerCPUKey, costPerCPU);
    localStorage.setItem(costPerGBKey, costPerGB);
    window.location.href = `${window.location.protocol}//${window.location.host}${window.location.pathname}`;
  }

  disableCostSettingsBtn.addEventListener("click", function () {
    clearData();
    window.location.href = window.location.href.split("?")[0];
  });

  function clearData() {
    localStorage.removeItem(emailEnteredKey);
    localStorage.removeItem(apiKeyLS);
    localStorage.removeItem(costPerCPUKey);
    localStorage.removeItem(costPerGBKey);
    localStorage.removeItem(selectedCloudProviderKey);
    localStorage.removeItem(selectedInstanceTypeKey);
  }
})();
