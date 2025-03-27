import Image from "next/image";

export default function Home() {
  return (
    <main className="relative min-h-screen flex flex-col justify-between font-[family-name:var(--font-geist-sans)]">
      <div className="w-full h-[54px] absolute top-[178px] border-y border-t-white/20 border-b-white/10 bg-white/[0.025]" />
      <div className="flex flex-row items-start w-full h-fit px-3">
        <Image
          className="dark:invert mt-16"
          src="/gs.svg"
          alt="GitSynth Logo"
          width={180}
          height={90}
          priority
        />
        <div className="flex flex-col border-x-2 border-white/10 h-screen relative ml-3">
          <h1 className="flex flex-row items-center justify-start gap-4 text-[10rem] font-medium -ml-3">
            GitSynth
          </h1>
          <p className="text-lg font-medium tracking-tight text-justify select-none -mt-3 mb-48 max-w-[640px] leading-snug">
            Version Control Systems like Git dominate top engineering team
            workflows. But Git is only an&nbsp;
            <span className="bg-gradient-to-r from-white/40 to-white/80 bg-clip-text text-transparent">
              approximation
            </span>{" "}
            of the underlying intent of a human author's changes, by comparing
            those changes across a lossy and conflict-prone medium: manifest
            lines of code. We're reimagining a tool that hasn't changed in over
            two decades to become{" "}
            <span className="bg-gradient-to-r from-red-500 to-purple-500 bg-clip-text text-transparent">
              intentionally and contextually aware
            </span>{" "}
            -- to capture the goal behind every keystroke, resolve conflicts
            ahead of time, and unlock fundamentally complex workflows for teams
            of any size for 10x development velocity at scale.
          </p>
          <div className="group absolute bottom-36 w-[640px] inline-block p-[2px] rounded-full bg-gradient-to-r from-red-500 to-purple-500 scale-105">
            <input
              autoFocus
              type="text"
              placeholder="https://github.com/your/repo/pull/1251"
              className="w-full rounded-full py-2 px-4 bg-gradient-to-b from-black to-[#0A0A0A] focus:outline-none z-10"
            />
          </div>
        </div>
        <ul className="flex flex-col h-[178px] relative gap-2 justify-end font-bold select-none">
          <li>HOME</li>
          <li>ABOUT</li>
          <li>JOBS</li>
        </ul>
      </div>
      <div className="w-full h-[54px] px-[204px] absolute bottom-[188px] border-t border-white/10 bg-white/[0.012] flex flex-row gap-3 items-center">
        <span className="px-2 py-1 rounded-full bg-gradient-to-r from-red-500 to-purple-500 text-white text-xs font-extrabold tracking-wider">
          PREVIEW
        </span>
        <span className="text-sm text-white tracking-tight font-extrabold">
          RESOLVE MERGE CONFLICTS ON ANY PR NOW
        </span>
      </div>
      <footer className="absolute border-t border-t-white/20 bottom-0 z-10 bg-[#0A0A0A] w-full h-fit px-3 flex flex-row flex-nowrap overflow-clip items-center justify-start">
        {new Array(8).fill(0).map((_, i) => (
          <Image
            key={i}
            className={`dark:invert opacity-10 -translate-x-${4 * (i === 0 ? 0 : i + 1)}`}
            src="/gs.svg"
            alt="GitSynth Logo"
            width={180}
            height={90}
            priority
          />
        ))}
      </footer>
    </main>
  );
}
