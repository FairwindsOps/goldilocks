(function () {
    const formId = "js-filter-form";
    const containerId = "js-filter-container";

    const form = document.getElementById(formId);
    const filterInput = form?.querySelector("input[type='search']");

    const container = document.getElementById(containerId);
    const potentialResults = container?.querySelectorAll("[data-filter]");
    const numPotentialResults = potentialResults?.length;

    function showFilterResult(result) {
        result.style.removeProperty("display");
    }

    function hideFilterResult(result) {
        result.style.display = "none";
    }

    function updateResults() {
        let filterTerm = filterInput.value;

        if (filterTerm) {
            let regex = new RegExp(`${ filterTerm.trim().replace(/\s/g, "|") }`, "i");

            for (let i = 0; i < numPotentialResults; i++) {
                let result = potentialResults[i];
                let filterWithin = result.dataset.filter;

                if (regex.test(filterWithin)) {
                    showFilterResult(result);
                } else {
                    hideFilterResult(result);
                }
            }
        } else {
            clearFilter();
        }
    }

    function clearFilter() {
        for (let i = 0; i < numPotentialResults; i++) {
            showFilterResult(potentialResults[i]);
        }
    }

    if (form && container) {
        if (numPotentialResults === 0) {
            form.style.display = "none";
            console.error("No filterable entries found, removed filter form");
        } else {
            filterInput.addEventListener("input", updateResults);

            form.addEventListener("submit", function(event) {
                event.preventDefault();
                updateResults();
            })
        }
    }
})();
