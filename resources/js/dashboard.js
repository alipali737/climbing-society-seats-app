window.onload = populateEventSelect;

async function fetchEventDetails() {
    const events = await fetch('/api/events')
    .then(response => {
        if (!response.ok) {
            throw new Error('Error fetching event details: Request failed with status ' + response.status);
        }
        return response.json();
    })
    .then(data => {
        return data;
    })
    .catch(error => {
        console.error(error);
    });
    return events;
}

async function fetchEventParticipants(eventId) {
    const participants = await fetch(`/api/participants?event=${eventId}`)
    .then(response => {
        if (!response.ok) {
            throw new Error('Error fetching event participants: Request failed with status ' + response.status)
        }
        return response.json();
    })
    .then(data => {
        return data;
    })
    .catch(error => {
        console.error(error);
    });
    return participants;
}

async function getEvents() {
    // Clear existing table data
    const tableBody = document.getElementById("event-table-body");
    tableBody.innerHTML = "";

    const fieldsToDisplay = ["session_location","session_date","meet_point","meet_time","total_seats","require_member","open_date","close_date"]

    try {
        // Get events from backend
        const events = await fetchEventDetails()

        // Populate the table with event data
        for (const event of events) {
            const row = document.createElement("tr");
            for (const key in event) {
                if (fieldsToDisplay.includes(key)) {
                    const cell = document.createElement("td");
                    cell.textContent = event[key];
                    row.appendChild(cell);
                }
            }
    
            const editCell = document.createElement("td");
            const editButton = document.createElement("button");
            editButton.textContent = "Edit";
            editButton.classList.add("warning-button");
            editButton.onclick = () => editEvent(row, event.event_id);
            editCell.appendChild(editButton);
    
            const deleteButton = document.createElement("button");
            deleteButton.textContent = "Delete";
            deleteButton.classList.add("danger-button");
            deleteButton.onclick = () => deleteEvent(event.event_id);
            editCell.appendChild(deleteButton);
    
            const linkButton = document.createElement("button");
            linkButton.textContent = "Link";
            linkButton.classList.add("info-button");
            linkButton.onclick = () => copyEventLink(event.event_id);
            editCell.appendChild(linkButton);

            row.appendChild(editCell);
            tableBody.appendChild(row);
        }
    } catch (error) {
        console.error(error);
    }

    populateEventSelect()
}

function copyEventLink(eventId) {
    const link = `${window.location.origin}/register?event=${eventId}`;
    navigator.clipboard.writeText(link)
        .then(() => {
            console.log("Link copied to clipboard:", link);
        })
        .catch(error => {
            console.error("Failed to copy link to clipboard:", error);
        });
    showCopyPopup(link);
}

function showCopyPopup(copyText) {
    var popup = document.getElementById("copyPopup");
    var popupText = document.getElementById("copyPopupText");

    popupText.innerHTML = "Copied <strong>" + copyText + "</strong> to clipboard!";
    popup.classList.add("show-popup");
    popup.classList.remove("hide-popup");
    setTimeout(hideCopyPopup, 3000);
}

function hideCopyPopup() {
    var popup = document.getElementById("copyPopup");
    popup.classList.remove("show-popup");
    popup.classList.add("hide-popup");
}

function editEvent(row, eventId) {
    const cells = row.getElementsByTagName("td");
    for (let i = 0; i < cells.length - 1; i++) {
        const cell = cells[i];
        const cellValue = cell.textContent;
        cell.innerHTML = `<input type="text" class="edit-cell" value="${cellValue}" />`;
    }

    const editCell = cells[cells.length - 1];
    editCell.innerHTML = `
        <button onclick="saveEvent(${row.rowIndex}, ${eventId})" class="success-button">Save</button>
        <button onclick="cancelEventEdit(${row.rowIndex})" class="danger-button">Cancel</button>
    `;
}

function cancelEventEdit(rowIndex) {
    getEvents();
}

function saveEvent(rowIndex, eventId) {
    const table = document.getElementById("event-table");
    const row = table.rows[rowIndex];
    const cells = row.getElementsByTagName("td");

    // Get edited values from input fields
    const editedEvent = {
        event_id: eventId,
        session_location: cells[0].querySelector('input').value,
        session_date: cells[1].querySelector('input').value,
        meet_point: cells[2].querySelector('input').value,
        meet_time: cells[3].querySelector('input').value,
        total_seats: parseInt(cells[4].querySelector('input').value),
        require_member: cells[5].querySelector('input').checked,
        open_date: cells[6].querySelector('input').value,
        close_date: cells[7].querySelector('input').value,
    };
    
    // POST API
    fetch('/api/events?event='+eventId, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(editedEvent)
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Error updating event: Request failed with status ' + response.status);
        }
        return response.json();
    })
    .then(data => {
        console.log(data.message);
        populateEventSelect();
        // Refresh the table with updated event data
        getEvents();
    })
    .catch(error => {
        console.error(error);
    });
}

const eventSelect = document.getElementById("event-select");

async function populateEventSelect() {

    clearOptions(eventSelect);
    eventSelect.selectElement = 0;

    try {
        // Get events from backend
        const events = await fetchEventDetails();
     
        // Populate the table with event data
        for (const event of events) {
             const option = document.createElement('option');
             option.value = event.event_id;
             option.textContent = event.session_date + ' - ' + event.session_location;
             eventSelect.appendChild(option);
        }
    } catch (error) {
        console.error(error);
    }
}

function clearOptions(selectElement) {
    var i, L = selectElement.options.length - 1;
    for(i = L; i >= 1; i--) {
        selectElement.remove(i);
    }
}

async function getParticipants() {
    const participantsTableBody = document.getElementById("participants-table-body")
    const selectedEventId = eventSelect.value;
    if (!selectedEventId) {
        // No event selected, clear table
        participantsTableBody.innerHTML = '';
        return;
    }

    try {
        // Get participants from backend
        const eventParticipants = await fetchEventParticipants(selectedEventId)
    
        // Populate table
        participantsTableBody.innerHTML = '';
        for (const participant of eventParticipants) {
            const row = document.createElement("tr");
            const firstNameCell = document.createElement("td");
            firstNameCell.textContent = participant.first_name;
            row.appendChild(firstNameCell);
    
            const lastNameCell = document.createElement("td");
            lastNameCell.textContent = participant.last_name;
            row.appendChild(lastNameCell);
    
            const deleteCell = document.createElement('td');
            const deleteButton = document.createElement('button');
            deleteButton.textContent = 'Delete';
            deleteButton.classList.add("danger-button");
            deleteButton.onclick = () => deleteParticipant(participant.participant_id);
            deleteCell.appendChild(deleteButton);
    
            row.appendChild(deleteCell);
    
            participantsTableBody.appendChild(row);
        }

        if (eventParticipants.length == 0) {
            const row = document.createElement("tr");
            const firstNameCell = document.createElement("td");
            firstNameCell.textContent = "No participants registered";
            row.appendChild(firstNameCell);
    
            const lastNameCell = document.createElement("td");
            lastNameCell.textContent = "No participants registered";
            row.appendChild(lastNameCell);

            const actionCell = document.createElement("td");
            row.appendChild(actionCell);
    
            participantsTableBody.appendChild(row);
        }

    } catch (error) {
        console.error(error);
    }
}

async function deleteEvent(eventId) {
    try {
        await fetchDeleteEvent(eventId);
        await new Promise(r => setTimeout(r, 500));
        populateEventSelect();
        getEvents();
    } catch (error) {
        console.error(error);
    }
}

async function fetchDeleteEvent(eventId) {
    fetch('/api/event?event='+eventId, {
        method: 'DELETE'
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Error deleting event: Request failed with status ' + response.status);
        }
        return response.json();
    })
    .then(data => {
        return data;
    })
    .catch(error => {
        console.error(error);
    });
}

async function deleteParticipant(participantId) {
    try {
        await fetchDeleteParticipant(participantId);
        await new Promise(r => setTimeout(r, 500));
        getParticipants();
    } catch (error) {
        console.error(error);
    }
}

async function fetchDeleteParticipant(participantId) {
    fetch('/api/participant?participant='+participantId, {
        method: 'DELETE'
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Error deleting participant: Request failed with status ' + response.status);
        }
        return response.json();
    })
    .then(data => {
        return data;
    })
    .catch(error => {
        console.error(error);
    });
}

// Create event
const createEventForm = document.getElementById("create-event-form");
createEventForm.addEventListener('submit', function (event) {
    event.preventDefault();
    const eventData = {
        session_location: document.getElementById('event_location').value,
        session_date: document.getElementById('event_date').value,
        meet_point: document.getElementById('meet_location').value,
        meet_time: document.getElementById('meet_time').value,
        total_seats: parseInt(document.getElementById('total_seats').value),
        require_member: document.getElementById('require_member').checked,
        open_date: document.getElementById('open_datetime').value,
        close_date: document.getElementById('close_datetime').value,
    };

    fetch('/api/events', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(eventData)
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Error creating event: Request failed with status ' + response.status);
        }
        return response.json();
    })
    .then(data => {
        responseText(data.message, data.success);
        populateEventSelect();
        createEventForm.reset();
    })
    .catch(error => {
        console.error(error);
    });
});

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