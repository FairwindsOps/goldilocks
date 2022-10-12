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

            form.addEventListener("submit", function(event) {
                event.preventDefault();
                updateResults();
            })
        }
    }

    function updateResults() {
        let filterTerm = filterInput.value;

        if (filterTerm) {
            let regex = new RegExp(`${ filterTerm.trim().replace(/\s/g, "|") }`, "i");

            for (result of potentialResults) {
                if (regex.test(result.dataset.filter)) {
                    showFilterResult(result);
                } else {
                    hideFilterResult(result);
                }
            }
        } else {
            clearFilter();
        }
    }

    function showFilterResult(result) {
        result.removeAttribute("hidden");
    }

    function hideFilterResult(result) {
        result.setAttribute("hidden", "");
    }

    function clearFilter() {
        for (result of potentialResults) {
            showFilterResult(result);
        }
    }
})();
