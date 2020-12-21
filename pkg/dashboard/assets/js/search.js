(function () {
    const formId = "js-search-form";
    const containerId = "js-search-container";

    const form = document.getElementById(formId);
    const searchInput = form?.querySelector("input[type='search']");

    const container = document.getElementById(containerId);
    const potentialResults = container?.querySelectorAll("[data-search]");
    const numPotentialResults = potentialResults?.length;

    function showSearchResult(result) {
        result.style.removeProperty("display");
    }

    function hideSearchResult(result) {
        result.style.display = "none";
    }

    function updateResults() {
        let searchTerm = searchInput.value;

        if (searchTerm) {
            let regex = new RegExp(`${ searchTerm.trim().replace(" ", "|") }`, "i");

            for (let i = 0; i < numPotentialResults; i++) {
                let result = potentialResults[i];
                let searchWithin = result.dataset.search;

                if (regex.test(searchWithin)) {
                    showSearchResult(result);
                } else {
                    hideSearchResult(result);
                }
            }
        } else {
            clearSearch();
        }
    }

    function clearSearch() {
        for (let i = 0; i < numPotentialResults; i++) {
            showSearchResult(potentialResults[i]);
        }
    }

    if (form && container) {
        if (numPotentialResults === 0) {
            form.style.display = "none";
            console.error("No filterable entries found, removed filter form");
        } else {
            searchInput.addEventListener("input", updateResults);
            
            form.addEventListener("submit", function(event) {
                event.preventDefault();
                updateResults();
            })
        }
    }
})();
