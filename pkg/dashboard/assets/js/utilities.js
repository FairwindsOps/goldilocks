function showElement(element) {
	element.removeAttribute("hidden");
}

function hideElement(element) {
	element.setAttribute("hidden", "");
}

export {
	showElement,
	hideElement
};
