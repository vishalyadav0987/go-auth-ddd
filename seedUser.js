// save as seedUsers.js
const fs = require('fs');

const URL = "http://localhost:8069/api/v1/auth/register";
const PASSWORD = "Vishal@123";

const baseEmail = "viahalyadav0987";
const domain = "@gmail.com";
const TOTAL_USERS = 1000;

const usedMobiles = new Set();
// Added mobile and password to CSV for better database tracking
let csvContent = "id,email,mobile,password\n";

function randomMobile() {
    while (true) {
        const num = Math.floor(6000000000 + Math.random() * 4000000000).toString();
        if (!usedMobiles.has(num)) {
            usedMobiles.add(num);
            return num;
        }
    }
}

function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

async function registerOne(i) {
    // Generate the Gmail alias instead of mailinator
    const email = `${baseEmail}+test${i}${domain}`;
    const mobile = randomMobile();

    const payload = { email, mobile, password: PASSWORD };

    try {
        const res = await fetch(URL, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(payload),
        });

        let data;
        try {
            data = await res.json();
        } catch {
            data = await res.text();
        }

        if (res.ok) {
            console.log(`✅ ${i} SUCCESS -> ${email} | ${mobile}`);
            // Record the successful registration for the CSV
            csvContent += `${i},${email},${mobile},${PASSWORD}\n`;
        } else {
            console.log(`❌ ${i} FAILED -> ${email} | ${res.status} |`, data);
        }
    } catch (err) {
        console.log(`🔥 ${i} ERROR -> ${err.message}`);
    }
}

async function registerUsers() {
    console.log(`🚀 Starting registration for ${TOTAL_USERS} users...`);

    for (let i = 1; i <= TOTAL_USERS; i++) {
        await registerOne(i);
        await sleep(100); // 100ms delay to prevent rate-limiting the local API
    }

    // Write the CSV file after all API calls finish
    fs.writeFileSync('test_users.csv', csvContent);
    console.log(`🎉 Done! Successfully generated test_users.csv with registered users`);
}

registerUsers();