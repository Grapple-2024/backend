export const disciplines = [
  {
    value: "gi-jiu-jitsu",
    label: "Gi Jiu Jitsu",
    backgroundColor: "#FFCCCC"
  },
  {
    value: "boxing",
    label: "Boxing",
    backgroundColor: "#FFDD99"
  },
  {
    value: "wrestling",
    label: "Wrestling",
    backgroundColor: "#FFFF99"
  },
  {
    value: "mma",
    label: "MMA",
    backgroundColor: "#99FF99"
  },
  {
    value: "karate",
    label: "Karate",
    backgroundColor: "#99FFFF"
  },
  {
    value: "muay-thai",
    label: "Muay Thai",
    backgroundColor: "#9999FF"
  },
  {
    value: "judo",
    label: "Judo",
    backgroundColor: "#FF99FF"
  },
  {
    value: "taekwondo",
    label: "Taekwondo",
    backgroundColor: "#FF6666"
  },
  {
    value: "no-gi-jiu-jitsu",
    label: "No Gi Jiu Jitsu",
    backgroundColor: "#FFCC99"
  }
];

export const difficulty = [
  {
    value: "Beginner",
    label: "Beginner"
  },
  {
    value: "Intermediate",
    label: "Intermediate"
  },
  {
    value: "Advanced",
    label: "Advanced"
  },
];

export const disciplineMapper = (discipline: string) => {
  return disciplines.find((d) => d.label === discipline)?.label;
}