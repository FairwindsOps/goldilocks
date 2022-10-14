(function () {
    const formId = "js-filter-form";
    const containerId = "js-filter-container";

    const form = document.getElementById(formId);
    const filterInput = form?.querySelector("input[type='text']");

    const container = document.getElementById(containerId);
    const potentialResults = container?.querySelectorAll("[data-filter]");
    const numPotentialResults = potentialResults?.length;

    let statusDelay = null;

    if (form && container) {
        if (numPotentialResults === 0) {
            form.setAttribute("hidden", "");
            console.error("No filterable entries found, removed filter form");
        } else {
            // Handle case where input value doesn't start empty (such as on page refresh)
            runFilter();

            filterInput.addEventListener("input", runFilter);
        }
    }

    function runFilter() {
        updateResults();
        updateStatus();
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
        const outputVisual = form.querySelector("output");
        const outputPolite = form.querySelector("output[aria-live='polite']");
        const outputAlert = form.querySelector("output[role='alert']");
        const numResults = container?.querySelectorAll("[data-filter]:not([hidden])").length;

        if (!filterInput.value) {
            outputVisual.textContent = `${numPotentialResults} namespaces found`;
        } else if (numResults === 0) {
            outputVisual.textContent = "No namespaces match filter";
        } else {
            outputVisual.textContent = `Showing ${numResults} out of ${numPotentialResults} namespaces`;
        }

        if (statusDelay) {
            window.clearTimeout(statusDelay);
        }

        /* 
           If you don't clear the content, then repeats of the same message aren't announced.
           There must be a time gap between clearing and injecting new content for this to work.
        */ 
        outputPolite.textContent = "";
        outputAlert.textContent = "";

        // Delay also helps make spoken announcements less disruptive by generating fewer of them
        statusDelay = window.setTimeout(() => {
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
        }, 1000);

    }
})();
