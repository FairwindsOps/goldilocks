function setJavascriptAvailable() {
	document.body.dataset.javascriptAvailable = true;
}

function showElement(element) {
	element.removeAttribute("hidden");
}

function hideElement(element) {
	element.setAttribute("hidden", "");
}

export {
	setJavascriptAvailable,
	showElement,
	hideElement
};
