// JavaScript file
function processData(data) {
    return data.map(item => ({
        ...item,
        processed: true
    }));
}

class APIManager {
    constructor(baseURL) {
        this.baseURL = baseURL;
    }

    async fetch(endpoint) {
        const response = await fetch(this.baseURL + endpoint);
        return response.json();
    }
}
