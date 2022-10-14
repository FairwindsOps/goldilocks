(function () {
    const formId = "js-filter-form";
    const containerId = "js-filter-container";

    const form = document.getElementById(formId);
    const filterInput = form?.querySelector("input[type='search']");

    const container = document.getElementById(containerId);
    const potentialResults = container?.querySelectorAll("[data-filter]");
    const numPotentialResults = potentialResults?.length;

    if (form && container) {
        if (numPotentialResults === 0) {
            form.setAttribute("hidden", "");
            console.error("No filterable entries found, removed filter form");
        } else {
            filterInput.addEventListener("input", updateResults);

        }
    }

    function updateResults() {
        let filterTerm = filterInput.value;

        if (filterTerm) {
            let regex = new RegExp(`${ filterTerm.trim().replace(/\s/g, "|") }`, "i");

            for (result of potentialResults) {
                if (regex.test(result.dataset.filter)) {
                    showElement(result);
                } else {
                    hideElement(result);
                }
            }
        } else {
            clearFilter();
        }

        updateStatus();
    }

    function showElement(element) {
        element.removeAttribute("hidden");
    }

    function hideElement(element) {
        element.setAttribute("hidden", "");
    }

    function clearFilter() {
        for (result of potentialResults) {
            showElement(result);
        }
    }

    function updateStatus() {
        const outputPolite = document.querySelector("output[aria-live='polite']");
        const outputAlert = document.querySelector("output[role='alert']");
        const numResults = container?.querySelectorAll("[data-filter]:not([hidden])").length;

        if (!filterInput.value) {
            outputPolite.textContent = `${numPotentialResults} namespaces found`;
            outputAlert.textContent = "";
        } else if (numResults === 0) {
            outputPolite.textContent = "";
            outputAlert.textContent = "No namespaces match filter";
        } else {
            outputPolite.textContent = `Showing ${numResults} out of ${numPotentialResults} namespaces`;
            outputAlert.textContent = "";
        }
    }
})();
