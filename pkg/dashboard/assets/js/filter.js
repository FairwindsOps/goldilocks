import {
    showElement,
    hideElement
} from "./utilities.js";

const form = document.getElementById("js-filter-form");
const container = document.getElementById("js-filter-container");

/* 
    These lookups simultaneously test that certain elements and attributes
    required for accessibility are present
*/
const filterInput = form?.querySelector("input[type='text']");
const potentialResults = container?.querySelectorAll("[data-filter]");

const outputVisual = form?.querySelector("output[aria-hidden]");
const outputPolite = form?.querySelector("output[aria-live='polite']");
const outputAlert = form?.querySelector("output[role='alert']");

let statusDelay = null;

// Test that all expected HTML is present
if (!form) {
    console.error("Could not find filter form");
} else if (!filterInput) {
    hideElement(form);
    console.error("Could not find filter input element, removed filter form");
} else if (!container) {
    hideElement(form);
    console.error("Could not find filter results container, removed filter form");
} else if (!outputVisual || !outputPolite || !outputAlert) {
    hideElement(form);
    console.error("Could not find all filter output elements, removed filter form");
} else if (potentialResults.length === 0) {
    hideElement(form);
    console.error("No filterable entries found, removed filter form");
} else {
    // HTML was successfully set up, wire in JS
    filterInput.addEventListener("input", runFilter);

    // Handle case where input value doesn't start empty (such as on page refresh)
    runFilter();
}

function runFilter() {
    updateResults();
    updateStatus();
}

function updateResults() {
    let filterTerm = filterInput.value;

    if (filterTerm) {
        let regex = new RegExp(`${ filterTerm.trim().replace(/\s/g, "|") }`, "i");

        for (const result of potentialResults) {
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

function clearFilter() {
    for (const result of potentialResults) {
        showElement(result);
    }
}

function updateStatus() {
    const numResults = container?.querySelectorAll("[data-filter]:not([hidden])").length;

    let message, type;

    if (!filterInput.value) {
        message = `${potentialResults.length} namespaces found`;
        type = "polite";
    } else if (numResults === 0) {
        message = "No namespaces match filter";
        type = "alert";
    } else {
        message = `Showing ${numResults} out of ${potentialResults.length} namespaces`;
        type = "polite";
    }

    changeStatusMessage(message, type);
}

function changeStatusMessage(message, type = "polite") {
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
