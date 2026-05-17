import { MarketingNav } from "@/components/marketing/nav";
import { Hero } from "@/components/marketing/hero";
import { Agents } from "@/components/marketing/agents";
import { How } from "@/components/marketing/how";
import { DashboardPreview } from "@/components/marketing/dashboard-preview";
import { Install } from "@/components/marketing/install";
import { Footer } from "@/components/marketing/footer";

export default function Landing() {
  return (
    <div className="min-h-screen flex flex-col">
      <MarketingNav />
      <main className="flex-1">
        <Hero />
        <How />
        <Agents />
        <DashboardPreview />
        <Install />
      </main>
      <Footer />
    </div>
  );
}
