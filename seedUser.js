// save as seedUsers.js

const URL = "http://localhost:8069/api/v1/auth/register";
const PASSWORD = "Vishal@123";

const usedMobiles = new Set();

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
    const email = `user${String(i).padStart(3, "0")}@mailinator.com`;
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
        } else {
            console.log(`❌ ${i} FAILED -> ${email} | ${res.status} |`, data);
        }
    } catch (err) {
        console.log(`🔥 ${i} ERROR -> ${err.message}`);
    }
}

async function registerUsers() {
    for (let i = 1; i <= 100; i++) {
        await registerOne(i);
        await sleep(100); // increase if rate limit
    }

    console.log("🎉 Done creating 100 users");
}

registerUsers();