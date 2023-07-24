var isLoginInProgress = false;

var form = document.getElementById('login-form');
form.addEventListener('submit', function (event) {
    event.preventDefault();

    if (!isLoginInProgress) {
        isLoginInProgress = true;
        loginLogic();
    }
})

const urlParams = new URLSearchParams(window.location.search);
const eventId = parseInt(urlParams.get('event'), 10);

const societyGreen = '#45B91A';
const bsDanger = '#DC3545';

async function loginLogic() {
    var button = document.getElementById('submit-button');
    var buttonContent = document.getElementById('submit-button-content');
    
    // Show spinner
    buttonContent.innerHTML = '<i class="fa fa-spinner fa-spin"></i> Loading...';
    button.disabled = true;

    // Make API Call
    var buttonContent = document.getElementById('submit-button-content');
    var username = form.elements['username'].value;
    var password = form.elements['password'].value;

    var jsonData = {
        username: username,
        password: password
    }

    try {
        const response = await fetch('/api/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(jsonData)
        });

        const data = await response.json();

        if (data.success) {
            buttonContent.innerHTML = '<i class="fa fa-check"></i> Success!';
            buttonContent.style.backgroundColor = societyGreen;
            responseText(data.message, true);

            setTimeout(() => {
                window.location.href = '/admin/dashboard.html';
            }, 1000);
        } else {
            buttonContent.innerHTML = '<i class="fa fa-times"></i> Error!';
            buttonContent.style.backgroundColor = bsDanger;
            responseText(data.message, false);
        }
    } catch (error) {
        // Show error symbol
        buttonContent.innerHTML = '<i class="fa fa-times"></i> Error!';
        buttonContent.style.backgroundColor = bsDanger;

        responseText(error, false);
        console.error(error);
    }

    setTimeout(() => {
        // Reset button
        buttonContent.innerHTML = 'login';
        buttonContent.style.backgroundColor = societyGreen;
        button.disabled = false;
        isLoginInProgress = false;
    }, 5000);
}

function responseText(text, success) {
    var displayElement = document.getElementById('response-text');
    displayElement.textContent = text;

    if (success) {
        displayElement.classList.remove('invalid-text');
        displayElement.classList.add('valid-text');
    } else {
        displayElement.classList.remove('valid-text');
        displayElement.classList.add('invalid-text');
    }

    setTimeout(() => {
        displayElement.textContent = '';
        displayElement.classList.remove('valid-text');
        displayElement.classList.remove('invalid-text');
    }, 5000);
}