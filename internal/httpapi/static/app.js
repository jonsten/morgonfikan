"use strict";

const dateFormatter = new Intl.DateTimeFormat("sv-SE", {
  weekday: "long",
  day: "numeric",
  month: "long",
  timeZone: "UTC",
});

function displayDate(value) {
  return dateFormatter.format(new Date(`${value}T12:00:00Z`));
}

function menuText(day) {
  if (day.served) {
    return day.items.join(", ");
  }
  return day.note ? `Ingen servering — ${day.note}` : "Ingen servering";
}

async function fetchJSON(path) {
  const response = await fetch(path, { headers: { Accept: "application/json" } });
  if (!response.ok) {
    throw new Error(`Tjänsten svarade med ${response.status}.`);
  }
  return response.json();
}

function renderToday(day) {
  document.querySelector("#today-date").textContent = displayDate(day.date);
  const menu = document.querySelector("#today-menu");
  menu.textContent = menuText(day);
  if (!day.served) {
    menu.classList.add("closed");
  }
}

function renderWeek(days) {
  const list = document.querySelector("#week-list");
  list.replaceChildren();
  for (const day of days) {
    const row = document.createElement("li");
    row.className = "week-day";

    const date = document.createElement("time");
    date.dateTime = day.date;
    date.textContent = displayDate(day.date);

    const serving = document.createElement("span");
    serving.className = day.served ? "week-menu" : "week-menu closed";
    serving.textContent = menuText(day);

    row.append(date, serving);
    list.append(row);
  }
}

async function loadFika() {
  const error = document.querySelector("#error");
  try {
    const [today, week] = await Promise.all([
      fetchJSON("/api/v1/today"),
      fetchJSON("/api/v1/week"),
    ]);
    renderToday(today);
    renderWeek(week.days);
  } catch (reason) {
    error.textContent = `Fikat kunde inte hämtas. ${reason.message}`;
    error.hidden = false;
    document.querySelector("#today-menu").textContent = "Uppgiften är inte tillgänglig just nu.";
  }
}

loadFika();
