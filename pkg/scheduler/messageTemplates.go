package scheduler

const EventOutputTemplate = `
## Event {{ .Event.EventID }} has closed! ##

Event Details:
- Event Location: {{ .Event.EventLocation }}
- Event Date: {{ .Event.EventDate }}
- Meet Location: {{ .Event.MeetLocation }}
- Meet Time: {{ .Event.MeetTime }}
- Seats Taken: {{ .Event.SeatsTaken }}/{{ .Event.TotalSeats }}
- Membership Required: {{ .Event.RequireMember }}

Participants:
{{- range $index, $participant := .Participants }}
{{ $participant.FirstName }} {{ $participant.LastName }}
{{- end }}
`
