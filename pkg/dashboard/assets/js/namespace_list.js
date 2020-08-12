$(function () {
    var $namespaceCards = $("#namespaceCards .card")
    var $searchBox = $("#search")

    // filters namespace cards containing the search text
    function filterNamespaceCards() {
        var searchText = $searchBox.val()
        $namespaceCards.filter(function() {
            var cardNamespace = $("strong", $(this)).text()
            var cardContainsSearchText = cardNamespace.toLowerCase().indexOf(searchText.toLowerCase()) > -1
            if(cardContainsSearchText) {
                $(this).show()
            } else {
                $(this).hide()
            }
        })
    }

    // clicks on the "view" button for the first visible namespace card
    function viewDashboardForFirstNamespace() {
        $namespaceCards.filter("div:not([style='display: none;'])").first().find("a[data-name='view'")[0].click()
    }

    // filter namespace cards on every key typed
    $searchBox.on("keyup", filterNamespaceCards)

    // start with a filter incase the browser caches input data
    filterNamespaceCards()
})
