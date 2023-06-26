var countDownDate = new Date("Sep 16, 2023 11:00:00").getTime();

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