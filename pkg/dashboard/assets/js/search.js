(function () {
    const formId = "js-search-form";
    const containerId = "js-search-container";

    const form = document.getElementById(formId);
    const searchInput = form.getElementsByTagName("input")[0];
    const submitBtn = form.querySelector("button[type='submit']");
    const container = document.getElementById(containerId);
    const potentialResults = container.children;
    const numPotentialResults = potentialResults.length;

    console.log(submitBtn);

    function updateResults() {
        let searchTerm = searchInput.value;

        if (searchTerm) {
            let showList = [];
            let hideList = [];
            let regex = new RegExp(`.*${ searchTerm }.*`, "i");

            for (let i = 0; i < numPotentialResults; i++) {
                let result = potentialResults[i];
                let searchWithin = result.dataset.search;

                if (regex.test(searchWithin)) {
                    showList.push(result);
                    result.style.removeProperty("display");
                } else {
                    hideList.push(result);
                    result.style.display = "none";
                }
            }

            console.log(showList);
            console.log(hideList);
        } else {
            clearSearch();
        }
    }

    function clearSearch() {
        console.log("todo");
    }

    searchInput.addEventListener("input", updateResults);
    submitBtn.addEventListener("submit", function(event) {
        console.log("submit event triggered");
        event.preventDefault();
        updateResults();
    })
})();
