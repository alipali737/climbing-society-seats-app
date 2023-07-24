
var form = document.getElementById('registerForm');
form.addEventListener('submit', function (event) {
    event.preventDefault();

    registerLogic();
    
})

const urlParams = new URLSearchParams(window.location.search);
const eventId = parseInt(urlParams.get('event'), 10);


const societyGreen = '#45B91A';
const bsDanger = '#DC3545';

async function registerLogic() {
    var button = document.getElementById('submit-button');
    var buttonContent = document.getElementById('submit-button-content');
    
    // Show spinner
    buttonContent.innerHTML = '<i class="fa fa-spinner fa-spin"></i> Loading...';
    button.disabled = true;

    // Make API Call
    await fetchRegisterAPI();

    setTimeout(() => {
        // Reset button
        buttonContent.innerHTML = 'register';
        buttonContent.style.backgroundColor = societyGreen;
        button.disabled = false;
        fetchEventDetails();
    }, 5000);
}

async function fetchRegisterAPI() {
    var buttonContent = document.getElementById('submit-button-content');
    var name = form.elements['name'].value;
    var member = document.getElementById('member').checked;

    var jsonData = {
        name: name,
        member: member,
        event: eventId
    }

    try {
        const response = await fetch('/api/register', {
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

// EVENT DETAILS SECTION

var countDownDate;

function fetchEventDetails() {
    fetch('/api/events?event='+eventId)
    .then(response => {
        if (!response.ok) {
            throw new Error('Error fetching event details: Request failed with status ' + response.status);
        }
        return response.json();
    })
    .then(data => {
        document.getElementById('session-location').textContent = data.session_location;
        document.getElementById('session-date').textContent = data.session_date;
        document.getElementById('meet-time').textContent = data.meet_time;
        document.getElementById('meet-point').textContent = data.meet_point;
        document.getElementById('current-seats').textContent = data.current_seats;
        document.getElementById('max-seats').textContent = data.total_seats;
        
        if (data.current_seats > 1) {
            document.getElementById('current-seats').classList.remove('invalid-text');
            document.getElementById('current-seats').classList.add('valid-text');
        } else {
            document.getElementById('current-seats').classList.remove('valid-text');
            document.getElementById('current-seats').classList.add('invalid-text');
            document.getElementById('submit-button').disabled = true;
        }

        countDownDate = convertToDate(data.close_date);
    })
    .catch(error => {
        const errorMessage = error.message;
        const errorPageUrl = '/register/error.html?message=' + encodeURIComponent(errorMessage);
        window.location.href = errorPageUrl;
    });
}

function convertToDate(dateString) {
    var parts = dateString.split(' ');
    var datePart = parts[0].split('/');
    var timePart = parts[1].split(':');
  
    var day = parseInt(datePart[0]);
    var month = parseInt(datePart[1]) - 1; // JavaScript months are zero-based
    var year = parseInt(datePart[2]);
    var hour = parseInt(timePart[0]);
    var minute = parseInt(timePart[1]);
    var second = parseInt(timePart[2]);
  
    return new Date(year, month, day, hour, minute, second);
  }

document.addEventListener('DOMContentLoaded', () => {
    fetchEventDetails(eventId);
})

// COUNTDOWN SECTION

const second = 1000,
    minute = second * 60,
    hour = minute * 60,
    day = hour * 24;

var x = setInterval(function() {
    var now = new Date().getTime();
    var distance = countDownDate - now;

    if (distance < 0) {
        clearInterval(x);
        document.getElementById("countdown-list").classList.add("disabled");
        document.getElementById("countdown-closed-text").classList.remove("disabled");
        document.getElementById('submit-button').disabled = true;
    }

    (document.getElementById("countdown-days").innerText = Math.floor(distance / day)),
        (document.getElementById("countdown-hours").innerText = Math.floor(
          (distance % day) / hour
        )),
        (document.getElementById("countdown-minutes").innerText = Math.floor(
          (distance % hour) / minute
        )),
        (document.getElementById("countdown-seconds").innerText = Math.floor(
          (distance % minute) / second
        ));
})
