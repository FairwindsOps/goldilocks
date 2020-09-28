(function () {
    const formId = "js-search-form";
    const containerId = "js-search-container";

    const form = document.getElementById(formId);
    const searchInput = form.getElementsByTagName("input")[0];
    const submitBtn = form.querySelector("button[type='submit']");
    const container = document.getElementById(containerId);
    const potentialResults = container.children;
    const numPotentialResults = potentialResults.length;

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

    searchInput.addEventListener("input", updateResults);
    
    form.addEventListener("submit", function(event) {
        event.preventDefault();
        updateResults();
    })
})();
