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

        let message, type;

        if (!filterInput.value) {
            message = `${numPotentialResults} namespaces found`;
            type = "polite";
        } else if (numResults === 0) {
            message = "No namespaces match filter";
            type = "alert";
        } else {
            message = `Showing ${numResults} out of ${numPotentialResults} namespaces`;
            type = "polite";
        }

        changeStatusMessage(message, type);
    }

    function changeStatusMessage(message, type = "polite") {
        const outputVisual = form.querySelector("output[aria-hidden]");
        const outputPolite = form.querySelector("output[aria-live='polite']");
        const outputAlert = form.querySelector("output[role='alert']");

        if (statusDelay) {
            window.clearTimeout(statusDelay);
        }

        outputVisual.textContent = message;
        outputPolite.textContent = "";
        outputAlert.textContent = "";

        /* 
           If you don't clear the content, then repeats of the same message aren't announced.
           There must be a time gap between clearing and injecting new content for this to work.
            Delay also:
            - Helps make spoken announcements less disruptive by generating fewer of them
            - Gives the screen reader a chance to finish announcing what's been typed, which will otherwise talk over these announcements (in MacOS/VoiceOver at least)
        */
        statusDelay = window.setTimeout(() => {
            switch (type) {
                case "polite":
                    outputPolite.textContent = message;
                    outputAlert.textContent = "";
                    break;
                case "alert":
                    outputPolite.textContent = "";
                    outputAlert.textContent = message;
                    break;
                default:
                    outputPolite.textContent = "Error: There was a problem with the filter.";
                    outputAlert.textContent = "";
            }
        }, 1000);
    }
})();
